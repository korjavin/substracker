'use strict';

// ---- Tab switching ----
document.querySelectorAll('.tab-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(s => s.classList.remove('active'));
    btn.classList.add('active');
    document.getElementById(btn.dataset.tab).classList.add('active');
    if (btn.dataset.tab === 'log') loadLog();
    if (btn.dataset.tab === 'notifications') loadNotifications();
  });
});

// ---- API helpers ----
async function api(method, path, body) {
  const opts = { method, headers: { 'Content-Type': 'application/json' } };
  if (body !== undefined) opts.body = JSON.stringify(body);
  const res = await fetch(path, opts);
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data;
}

// ---- Date helpers ----
function nextResetDate(billingDay) {
  const now = new Date();
  const year = now.getFullYear();
  const month = now.getMonth();
  const today = now.getDate();

  let candidate = new Date(year, month, billingDay);
  if (candidate <= now) {
    candidate = new Date(year, month + 1, billingDay);
  }
  return candidate;
}

function daysUntil(date) {
  const now = new Date();
  now.setHours(0, 0, 0, 0);
  date.setHours(0, 0, 0, 0);
  return Math.round((date - now) / 86400000);
}

function formatDate(date) {
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

function serviceBadge(service) {
  const map = { claude: 'claude', googleone: 'googleone', openai: 'openai', zai: 'zai' };
  const labels = { claude: 'Claude', googleone: 'Google One', openai: 'OpenAI', zai: 'Z.ai', other: 'Other' };
  const cls = map[service] || 'other';
  const label = labels[service] || (service.charAt(0).toUpperCase() + service.slice(1));
  return `<span class="badge badge-${cls}">${label}</span>`;
}

// ---- Usage Status ----
async function loadUsage() {
  const el = document.getElementById('usage-content');
  try {
    const data = await api('GET', '/api/providers/usage/cached');
    renderUsage(data);
  } catch (e) {
    if (e.message.includes('no_cached_usage')) {
      el.innerHTML = `<p style="font-size:13px;color:var(--text-dim)">No usage data yet. Click Refresh to fetch.</p>`;
    } else {
      el.innerHTML = `<p style="font-size:13px;color:var(--red)">${e.message}</p>`;
    }
  }
}

async function refreshUsage() {
  const btn = document.getElementById('refresh-usage-btn');
  const el = document.getElementById('usage-content');
  btn.disabled = true;
  el.innerHTML = `<p style="font-size:13px;color:var(--text-dim)">Fetching from provider...</p>`;
  try {
    const data = await api('GET', '/api/providers/claude/usage');
    renderUsage({
      provider_name: 'Claude',
      current_usage_seconds: data.CurrentUsageSeconds || 0,
      total_limit_seconds: data.TotalLimitSeconds || 0,
      is_blocked: data.IsBlocked || false,
      fetched_at: new Date().toISOString(),
      session_usage_pct: data.session_usage_pct,
      session_resets_at: data.session_resets_at,
      weekly_usage_pct: data.weekly_usage_pct,
      weekly_resets_at: data.weekly_resets_at
    });
  } catch (e) {
    if (e.message.includes('relogin_required')) {
      el.innerHTML = `<p style="font-size:13px;color:var(--yellow)">Login required. <a href="#" onclick="openSettingsModal(); return false;" style="color:var(--accent);">Go to settings</a> to update your session key.</p>`;
    } else {
      el.innerHTML = `<p style="font-size:13px;color:var(--red)">Error: ${e.message}</p>`;
    }
  } finally {
    btn.disabled = false;
  }
}

document.getElementById('refresh-usage-btn').addEventListener('click', refreshUsage);
document.getElementById('settings-btn').addEventListener('click', openSettingsModal);

// ---- Settings Modal ----
const settingsBackdrop = document.getElementById('settings-backdrop');
const settingsForm = document.getElementById('settings-form');

function openSettingsModal() {
  settingsBackdrop.style.display = 'flex';
  document.getElementById('settings-session-key').focus();
}

function closeSettingsModal() {
  settingsBackdrop.style.display = 'none';
  settingsForm.reset();
}

document.getElementById('settings-cancel').addEventListener('click', closeSettingsModal);
settingsBackdrop.addEventListener('click', e => { if (e.target === settingsBackdrop) closeSettingsModal(); });

settingsForm.addEventListener('submit', async e => {
  e.preventDefault();
  const sessionKey = document.getElementById('settings-session-key').value.trim();
  const btn = document.getElementById('settings-save');
  btn.disabled = true;
  try {
    await api('POST', '/api/providers/claude/login', { session_key: sessionKey });
    closeSettingsModal();
    refreshUsage();
  } catch (err) {
    alert('Failed to save session key: ' + err.message);
  } finally {
    btn.disabled = false;
  }
});

function renderUsage(u) {
  const el = document.getElementById('usage-content');
  const percent = u.total_limit_seconds > 0 ? Math.min(100, (u.current_usage_seconds / u.total_limit_seconds) * 100) : 0;

  // If we have detailed Claude limits (session/weekly pct)
  if (u.session_usage_pct !== undefined && u.weekly_usage_pct !== undefined) {
    const sessionPct = Math.min(100, Math.round(u.session_usage_pct * 100));
    const weeklyPct = Math.min(100, Math.round(u.weekly_usage_pct * 100));
    const isSessionDanger = sessionPct > 90 || u.is_blocked;
    const isWeeklyDanger = weeklyPct > 90 || u.is_blocked;

    const sResets = new Date(u.session_resets_at);
    const wResets = new Date(u.weekly_resets_at);

    const blockedBadge = u.is_blocked ? `<span class="badge-blocked">BLOCKED</span>` : '';
    const d = new Date(u.fetched_at);

    el.innerHTML = `
      <div style="font-size:14px; font-weight: 500; margin-bottom: 8px;">
        ${esc(u.provider_name)} ${blockedBadge}
      </div>

      <div style="font-size:13px; color:var(--text); margin-bottom:4px;">Current session (${sessionPct}%)</div>
      <div class="usage-bar-container">
        <div class="usage-bar ${isSessionDanger ? 'danger' : ''}" style="width: ${sessionPct}%"></div>
      </div>
      <div class="usage-details" style="margin-bottom:12px">
        <span>Resets in: ${sResets.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}</span>
      </div>

      <div style="font-size:13px; color:var(--text); margin-bottom:4px;">Weekly limit (${weeklyPct}%)</div>
      <div class="usage-bar-container">
        <div class="usage-bar ${isWeeklyDanger ? 'danger' : ''}" style="width: ${weeklyPct}%"></div>
      </div>
      <div class="usage-details">
        <span>Resets: ${formatDate(wResets)} ${wResets.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}</span>
        <span>Checked: ${d.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}</span>
      </div>
    `;
    return;
  }

  // Claude's API might not provide exact numbers, but gives IsBlocked flag.
  // If we only have IsBlocked flag, just show status.
  if (u.total_limit_seconds === 0 && u.current_usage_seconds === 0) {
    const statusText = u.is_blocked
      ? `<span style="color:var(--red);font-weight:600">BLOCKED</span> (Limit exceeded)`
      : `<span style="color:var(--green);font-weight:600">ACTIVE</span>`;

    const d = new Date(u.fetched_at);
    el.innerHTML = `
      <div style="font-size:14px; margin-bottom:8px;">
        <strong>${esc(u.provider_name)} Status:</strong> ${statusText}
      </div>
      <div class="usage-details">
        <span>Last checked: ${d.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}</span>
      </div>
    `;
    return;
  }

  const isDanger = percent > 90 || u.is_blocked;
  const barClass = isDanger ? 'usage-bar danger' : 'usage-bar';
  const blockedBadge = u.is_blocked ? `<span class="badge-blocked">BLOCKED</span>` : '';

  const currentHours = (u.current_usage_seconds / 3600).toFixed(1);
  const totalHours = (u.total_limit_seconds / 3600).toFixed(1);

  const d = new Date(u.fetched_at);

  el.innerHTML = `
    <div style="font-size:14px; font-weight: 500;">
      ${esc(u.provider_name)} ${blockedBadge}
    </div>
    <div class="usage-bar-container">
      <div class="${barClass}" style="width: ${percent}%"></div>
    </div>
    <div class="usage-details">
      <span>${currentHours}h / ${totalHours}h used</span>
      <span>Last checked: ${d.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}</span>
    </div>
  `;
}

// Poll usage every 5 mins
setInterval(loadUsage, 5 * 60 * 1000);

// ---- Subscriptions ----
let subs = [];

async function loadSubs() {
  try {
    subs = await api('GET', '/api/subscriptions');
    renderSubs();
  } catch (e) {
    document.getElementById('subs-container').innerHTML =
      `<div class="empty"><div>Error: ${e.message}</div></div>`;
  }
}

function renderSubs() {
  const el = document.getElementById('subs-container');
  if (!subs.length) {
    el.innerHTML = `<div class="empty">
      <div style="font-size:32px">📋</div>
      <p>No subscriptions yet. Add one to get started.</p>
    </div>`;
    return;
  }

  const items = subs.map(s => {
    const nextDate = nextResetDate(s.billing_day);
    const days = daysUntil(nextDate);
    let resetCell;
    if (days === 0) {
      resetCell = `<span class="reset-today">Today!</span>`;
    } else if (days <= 3) {
      resetCell = `<span class="reset-soon">${formatDate(nextDate)} (${days}d)</span>`;
    } else {
      resetCell = `<span style="color:var(--text-dim)">${formatDate(nextDate)} (${days}d)</span>`;
    }

    const rowHtml = `<tr onclick="openDetail(${s.id})" style="cursor:pointer">
      <td><strong>${esc(s.name)}</strong></td>
      <td>${serviceBadge(s.service)}</td>
      <td style="color:var(--text-dim)">Day ${s.billing_day}</td>
      <td>${resetCell}</td>
    </tr>`;

    const cardHtml = `<div class="sub-card" onclick="openDetail(${s.id})" style="cursor:pointer">
      <div class="sub-card-header">
        <div>
          <div class="sub-card-title">${esc(s.name)}</div>
        </div>
        ${serviceBadge(s.service)}
      </div>
      <div class="sub-card-details">
        <div>Reset Day: ${s.billing_day}</div>
        <div>Next Reset: ${resetCell}</div>
      </div>
    </div>`;

    return { rowHtml, cardHtml };
  });

  const rows = items.map(i => i.rowHtml).join('');
  const cards = items.map(i => i.cardHtml).join('');

  el.innerHTML = `${cards}<table>
    <thead><tr>
      <th>Name</th><th>Service</th><th>Reset Day</th><th>Next Reset</th>
    </tr></thead>
    <tbody>${rows}</tbody>
  </table>`;
}

function esc(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

async function deleteSub(id) {
  if (!confirm('Delete this subscription?')) return;
  try {
    await api('DELETE', `/api/subscriptions/${id}`);
    await loadSubs();
  } catch (e) {
    alert('Error: ' + e.message);
  }
}

// ---- Detail Panel ----
const detailPanel = document.getElementById('detail-panel');
const detailBackdrop = document.getElementById('detail-backdrop');
let currentDetailId = null;

function openDetail(id) {
  const sub = subs.find(s => s.id === id);
  if (!sub) return;
  currentDetailId = id;

  document.getElementById('detail-title').textContent = sub.name;
  document.getElementById('detail-service').innerHTML = serviceBadge(sub.service);
  document.getElementById('detail-day').textContent = sub.billing_day;

  const createdDate = new Date(sub.created_at || Date.now());
  document.getElementById('detail-created').textContent = createdDate.toLocaleString();

  const notesContainer = document.getElementById('detail-notes-container');
  const notesEl = document.getElementById('detail-notes');
  if (sub.notes) {
    notesEl.textContent = sub.notes;
    notesContainer.style.display = 'block';
  } else {
    notesContainer.style.display = 'none';
  }

  detailBackdrop.style.display = 'block';
  setTimeout(() => detailPanel.classList.add('open'), 10);
}

function closeDetail() {
  detailPanel.classList.remove('open');
  setTimeout(() => {
    detailBackdrop.style.display = 'none';
    currentDetailId = null;
  }, 200);
}

document.getElementById('detail-close').addEventListener('click', closeDetail);
detailBackdrop.addEventListener('click', closeDetail);

document.getElementById('detail-edit-btn').addEventListener('click', () => {
  if (currentDetailId) openEdit(currentDetailId);
});

document.getElementById('detail-delete-btn').addEventListener('click', () => {
  if (currentDetailId) {
    deleteSub(currentDetailId).then(() => {
      closeDetail();
    });
  }
});

// ---- Modal ----
const backdrop = document.getElementById('modal-backdrop');
const form = document.getElementById('sub-form');

document.getElementById('add-sub-btn').addEventListener('click', () => openModal());
document.getElementById('modal-cancel').addEventListener('click', closeModal);
backdrop.addEventListener('click', e => { if (e.target === backdrop) closeModal(); });

function openModal(sub) {
  document.getElementById('modal-title').textContent = sub ? 'Edit Subscription' : 'Add Subscription';
  document.getElementById('sub-id').value = sub ? sub.id : '';
  document.getElementById('sub-name').value = sub ? sub.name : '';
  document.getElementById('sub-service').value = sub ? sub.service : 'claude';
  document.getElementById('sub-day').value = sub ? sub.billing_day : '';
  document.getElementById('sub-notes').value = sub ? sub.notes : '';
  backdrop.classList.add('open');
  document.getElementById('sub-name').focus();
}

function openEdit(id) {
  const sub = subs.find(s => s.id === id);
  if (sub) openModal(sub);
}

function closeModal() {
  backdrop.classList.remove('open');
  form.reset();
}

form.addEventListener('submit', async e => {
  e.preventDefault();
  const id = document.getElementById('sub-id').value;
  const payload = {
    name: document.getElementById('sub-name').value.trim(),
    service: document.getElementById('sub-service').value,
    billing_day: parseInt(document.getElementById('sub-day').value, 10),
    notes: document.getElementById('sub-notes').value.trim(),
  };
  const saveBtn = document.getElementById('modal-save');
  saveBtn.disabled = true;
  try {
    if (id) {
      await api('PUT', `/api/subscriptions/${id}`, payload);
    } else {
      await api('POST', '/api/subscriptions', payload);
    }
    closeModal();
    await loadSubs();
    if (currentDetailId) {
      openDetail(currentDetailId);
    }
  } catch (e) {
    alert('Error: ' + e.message);
  } finally {
    saveBtn.disabled = false;
  }
});

// ---- Notifications ----
let pushSubscription = null;

async function loadNotifications() {
  await Promise.all([loadTelegramChats(), initPushButton()]);
}

async function initPushButton() {
  const btn = document.getElementById('push-btn');
  const msg = document.getElementById('push-status-msg');

  if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
    msg.style.display = 'block';
    msg.className = 'alert alert-warn';
    msg.textContent = 'Web Push is not supported in this browser.';
    btn.style.display = 'none';
    return;
  }

  try {
    const resp = await api('GET', '/api/vapid-public-key');
    if (!resp.key) {
      msg.style.display = 'block';
      msg.className = 'alert alert-warn';
      msg.textContent = 'VAPID keys not configured on server. Web Push is disabled.';
      btn.style.display = 'none';
      return;
    }

    const reg = await navigator.serviceWorker.register('/sw.js');
    pushSubscription = await reg.pushManager.getSubscription();

    if (pushSubscription) {
      btn.textContent = 'Disable Push Notifications';
      btn.classList.add('subscribed');
    } else {
      btn.textContent = 'Enable Push Notifications';
      btn.classList.remove('subscribed');
    }

    btn.onclick = async () => {
      btn.disabled = true;
      try {
        if (pushSubscription) {
          await pushSubscription.unsubscribe();
          await api('DELETE', '/api/webpush/subscribe', { endpoint: pushSubscription.endpoint });
          pushSubscription = null;
          btn.textContent = 'Enable Push Notifications';
          btn.classList.remove('subscribed');
        } else {
          const appKey = urlBase64ToUint8Array(resp.key);
          pushSubscription = await reg.pushManager.subscribe({
            userVisibleOnly: true,
            applicationServerKey: appKey,
          });
          await api('POST', '/api/webpush/subscribe', pushSubscription.toJSON());
          btn.textContent = 'Disable Push Notifications';
          btn.classList.add('subscribed');
        }
      } catch (e) {
        alert('Error: ' + e.message);
      } finally {
        btn.disabled = false;
      }
    };
  } catch (e) {
    msg.style.display = 'block';
    msg.className = 'alert alert-warn';
    msg.textContent = 'Push setup failed: ' + e.message;
  }
}

function urlBase64ToUint8Array(base64) {
  const padding = '='.repeat((4 - base64.length % 4) % 4);
  const b64 = (base64 + padding).replace(/-/g, '+').replace(/_/g, '/');
  const raw = atob(b64);
  const arr = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) arr[i] = raw.charCodeAt(i);
  return arr;
}

// ---- Telegram ----
async function loadTelegramChats() {
  try {
    const chats = await api('GET', '/api/telegram/chats');
    renderTelegramChats(chats);
  } catch (e) {
    document.getElementById('tg-chats-list').innerHTML = `<p style="color:var(--red)">${e.message}</p>`;
  }
}

function renderTelegramChats(chats) {
  const el = document.getElementById('tg-chats-list');
  if (!chats.length) {
    el.innerHTML = `<p style="font-size:13px;color:var(--text-dim);padding-top:8px">No Telegram chats added yet.</p>`;
    return;
  }
  el.innerHTML = chats.map(c => `
    <div class="chat-item">
      <span class="chat-id">${esc(c.chat_id)}</span>
      <button class="btn btn-danger btn-sm" onclick="removeTelegramChat('${esc(c.chat_id)}')">Remove</button>
    </div>`).join('');
}

document.getElementById('tg-add-btn').addEventListener('click', async () => {
  const input = document.getElementById('tg-chat-input');
  const chatID = input.value.trim();
  if (!chatID) return;
  try {
    await api('POST', '/api/telegram/chats', { chat_id: chatID });
    input.value = '';
    await loadTelegramChats();
  } catch (e) {
    alert('Error: ' + e.message);
  }
});

async function removeTelegramChat(chatID) {
  if (!confirm(`Remove chat ${chatID}?`)) return;
  try {
    await api('DELETE', `/api/telegram/chats/${encodeURIComponent(chatID)}`);
    await loadTelegramChats();
  } catch (e) {
    alert('Error: ' + e.message);
  }
}

document.getElementById('test-notif-btn').addEventListener('click', async () => {
  const btn = document.getElementById('test-notif-btn');
  const result = document.getElementById('test-result');
  btn.disabled = true;
  result.textContent = 'Sending...';
  try {
    await api('POST', '/api/notifications/test');
    result.textContent = 'Sent! Check your configured channels.';
    result.style.color = 'var(--green)';
  } catch (e) {
    result.textContent = 'Error: ' + e.message;
    result.style.color = 'var(--red)';
  } finally {
    btn.disabled = false;
    setTimeout(() => { result.textContent = ''; }, 5000);
  }
});

// ---- Log ----
async function loadLog() {
  const el = document.getElementById('log-container');
  try {
    const logs = await api('GET', '/api/notifications/log');
    if (!logs.length) {
      el.innerHTML = `<div class="empty">
        <div style="font-size:32px">📭</div>
        <p>No notifications sent yet.</p>
      </div>`;
      return;
    }
    el.innerHTML = logs.map(l => {
      const d = new Date(l.sent_at);
      const timeStr = isNaN(d) ? l.sent_at : d.toLocaleString();
      const chClass = `log-channel-${l.channel}`;
      return `<div class="log-item">
        <span class="log-time">${timeStr}</span>
        <span class="log-channel ${chClass}">${l.channel}</span>
        <span class="log-msg">${esc(l.message)}</span>
      </div>`;
    }).join('');
  } catch (e) {
    el.innerHTML = `<div class="empty"><div>Error: ${e.message}</div></div>`;
  }
}

document.getElementById('refresh-log-btn').addEventListener('click', loadLog);

// ---- Init ----
loadSubs();
loadUsage();
