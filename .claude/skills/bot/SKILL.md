---
name: bot
description: Build chat bots in a togo app — register cross-platform commands/handlers with the bot plugin (Telegram/Discord/Slack)
---

# togo bot

The `bot` plugin gives a togo app a **chat bot** that answers commands across
Telegram, Discord and Slack from one set of handlers.

## Setup

```bash
togo install togo-framework/bot
togo install togo-framework/bot-telegram   # or bot-discord / bot-slack
```

`.env`:

```bash
BOT_DRIVER=telegram
TELEGRAM_BOT_TOKEN=...        # bot-telegram
# DISCORD_BOT_TOKEN=...       # bot-discord
# SLACK_APP_TOKEN=xapp-...    # bot-slack (Socket Mode app token)
# SLACK_BOT_TOKEN=xoxb-...    # bot-slack
```

## Register handlers

Add a file like `internal/bot/handlers.go` and blank-import it from `main`:

```go
package bot

import (
	"context"
	tbot "github.com/togo-framework/bot"
)

func init() {
	tbot.OnCommand("ping", func(ctx context.Context, b *tbot.Service, m tbot.Message) (string, error) {
		return "pong 🏓", nil
	})
	tbot.OnCommand("echo", func(ctx context.Context, b *tbot.Service, m tbot.Message) (string, error) {
		return m.Text, nil
	})
}
```

- `OnCommand("name", h)` runs `h` when a message is `/name ...`. `m.Args` holds
  the whitespace-split arguments.
- `OnMessage(h)` runs `h` for every non-command message.
- Return a non-empty string to reply; return `("", nil)` to stay silent.

## Send proactively

```go
if svc, ok := tbot.FromKernel(k); ok {
	_ = svc.Send(ctx, channelID, "build finished ✅")
}
```

## Rules

- One bot driver is active per process (`BOT_DRIVER`). Handlers are
  platform-agnostic — don't switch on `m.Platform` unless you must.
- Never hard-code tokens; read them from env (the driver already does).
- Keep handlers fast; offload slow work to the `queue` plugin and reply later
  via `Service.Send`.
