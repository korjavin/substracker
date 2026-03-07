# SubsTracker

A small helper to track usage for different Claude/OpenAI/Z.ai subscriptions and notify users when limits reset.

## Features

- Track subscription billing cycles (Claude, OpenAI, Z.ai, and others)
- Automatic daily check for resets — notifies on reset day and 1 day before
- **Web Push notifications** (VAPID/browser push)
- **Telegram notifications** (bot sends messages to configured chat IDs)
- Clean web UI for managing subscriptions and notification settings
- SQLite database (pure Go, no CGO)

## Quick Start

```bash
cp .env.example .env
# Edit .env with your settings
./start.sh
```

Open http://localhost:8080

## Environment Variables

| Variable | Description | Required |
|---|---|---|
| `PORT` | HTTP port (default: `8080`) | No |
| `DB_PATH` | SQLite file path (default: `data.db`) | No |
| `TG_BOT_TOKEN` | Telegram bot token from @BotFather | Optional |
| `VAPID_PUBLIC_KEY` | VAPID public key for web push | Optional |
| `VAPID_PRIVATE_KEY` | VAPID private key for web push | Optional |
| `VAPID_SUBJECT` | VAPID subject, e.g. `mailto:you@example.com` | If VAPID set |

## Generating VAPID Keys

```bash
npx web-push generate-vapid-keys
```

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
