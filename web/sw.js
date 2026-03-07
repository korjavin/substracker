self.addEventListener('push', function (event) {
  const data = event.data ? event.data.json() : {};
  const title = 'SubsTracker';
  const options = {
    body: data.message || 'Subscription update',
    icon: '/favicon.ico',
  };
  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener('notificationclick', function (event) {
  event.notification.close();
  event.waitUntil(clients.openWindow('/'));
});
