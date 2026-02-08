# Telegram Chat Bot

A Telegram bot for running a daily "roulette" game in group chats. 
Each day, one random participant is selected as the winner.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TELEGRAM_BOT_TOKEN` | Yes | - | Bot token from BotFather |
| `DB_PATH` | No | `bot.db` | Path to SQLite database file |
| `ROLL_COMMAND` | No | `roll` | Command name to trigger the roulette (without `/`) |
| `ADMIN_IDS` | No | _(empty)_ | Comma-separated Telegram user IDs allowed to use `/reset`. When empty, `/reset` is available to everyone. |

## Database Setup

The bot automatically creates the database schema on first run. To seed translations and message sets:

```bash
sqlite3 bot.db < init-demo-data.sql
```

## Running

```bash
TELEGRAM_BOT_TOKEN="your-bot-token" ./telegram-chat-bot
```

## Commands

| Command | Description |
|---------|-------------|
| `/join` | Join the roulette game |
| `/roll` | Spin the roulette |
| `/stats` | Show win statistics |
| `/participants` | List all participants |

## Customization

### Translations

All bot messages are stored in the `translations` table and can be customized directly in the database.

### Message Sets

The roulette announcement uses random message sets from the database. 
Each set contains multiple messages sent in sequence with the final message announcing the winner. 
Add custom sets to the `message_sets` and `set_messages` tables.

## Development

### sqlc

Database queries are managed with [sqlc](https://sqlc.dev/). 
The `db/` package is entirely generated - never edit files in `db/` by hand.

```bash
# Regenerate db/ after editing schema.sql or queries.sql
sqlc generate
```
