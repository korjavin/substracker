# SubsTracker

A small helper to track usage for different Claude/OpenAI/Z.ai subscriptions and notify users when limits reset.

## Features

- Track subscription billing cycles (Claude, OpenAI, Z.ai, and others)
- Automatic daily check for resets — notifies on reset day and 1 day before
- **Web Push notifications** (VAPID/browser push)
- **Telegram notifications** (bot sends messages to configured chat IDs)
- Clean web UI for managing subscriptions and notification settings
- SQLite database (pure Go, no CGO)

## Tracked Providers
Currently, SubsTracker supports automated usage tracking for:
- **Claude (Anthropic)**: requires a `session_key` cookie from claude.ai
- **Google One**: requires a `SID` cookie from one.google.com
- **Z.ai**: requires a `session_cookie` cookie from z.ai

To enable tracking, open the settings menu (gear icon) next to the provider in the web interface and enter the required cookie.

## Quick Start

```bash
cp .env.example .env
# Edit .env with your settings
./start.sh
```

Open http://localhost:5454

## Environment Variables

| Variable | Description | Required |
|---|---|---|
| `PORT` | HTTP port (default: `5454`) | No |
| `DB_PATH` | SQLite file path (default: `data.db`) | No |
| `TG_BOT_TOKEN` | Telegram bot token from @BotFather | Optional |
| `VAPID_PUBLIC_KEY` | VAPID public key for web push | Optional |
| `VAPID_PRIVATE_KEY` | VAPID private key for web push | Optional |
| `VAPID_SUBJECT` | VAPID subject, e.g. `mailto:you@example.com` | If VAPID set |
| `SESSION_SECRET` | Secret key for signing session cookies | Yes |
| `TELEGRAM_BOT_USERNAME` | Telegram bot username for Login Widget | Yes |
| `QUOTA_POLL_INTERVAL` | Interval to check provider usage quota | Optional (default: `15m`) |

## Generating VAPID Keys

```bash
npx web-push generate-vapid-keys
```

## Google One Login
If you are tracking Google One storage, you need to provide your session cookie. In the web interface, click the Settings gear icon in the Google One usage block, and enter your `SID` cookie value (or a full cookie string containing `SID`, `HSID`, and `SSID`). You can find this by opening Developer Tools -> Application -> Cookies on `one.google.com`.

## Telegram Setup

1. Create a bot via [@BotFather](https://t.me/BotFather) — get a bot token
2. Set `TG_BOT_TOKEN` in `.env`
3. Get your Telegram chat ID from [@userinfobot](https://t.me/userinfobot)
4. Add your chat ID in the web UI under **Notifications**

## Docker

```bash
docker compose up -d
```

## Deployment (Portainer)

See `.github/workflows/deploy.yml`. The workflow:
1. Builds and pushes image to GHCR on `master` push
2. Updates the `deploy` branch with the new image tag
3. Triggers a Portainer webhook

Set these GitHub secrets: `PORTAINER_WEBHOOK` (and `PORTAINER_WEBHOOK_DEV` for dev)
