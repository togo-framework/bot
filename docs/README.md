# bot — docs

**Bot.** Build chat bots — one command/handler registry, swappable Discord/Slack/Telegram drivers.

## Install

```bash
togo install togo-framework/bot
```

Select a driver via **BOT_DRIVER (or bot.provider)**. Drive it from the CLI with **`togo bot`**.

## Interface

`Bot` — `Start`/`Stop`/`Send`, plus a command/handler registry (`OnCommand`/`OnMessage`) so any plugin can add bot commands.

## Configuration

| Env var | Description |
|---|---|
| `BOT_DRIVER` | Selects the bot driver (alternative to togo.yaml `bot.provider`). |

## Usage & notes

Register commands from any plugin: `bot.OnCommand("/foo", handler)` / `bot.OnMessage(handler)`. The kernel starts the configured `BOT_DRIVER` on Boot; `togo bot:send` / `bot:ask` drive it from the CLI.

## Example

```bash
togo bot:send '#general' 'Deployed!'
togo bot:ask 'summarize the latest release'
```

## Links

- [Driver plugins](https://to-go.dev/marketplace)
- [Marketplace](https://to-go.dev/marketplace)
- [Source](https://github.com/togo-framework/bot)
