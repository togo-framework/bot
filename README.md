<!-- togo-header -->
<div align="center">
  <picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/togo-mark-dark.svg" /><img src=".github/assets/togo-mark.svg" alt="ToGO" height="64" /></picture>
  <h1>togo-framework/bot</h1>
  <p>
    <a href="https://to-go.dev/marketplace"><img src="https://img.shields.io/badge/marketplace-to--go.dev-1F8A99" alt="marketplace" /></a>
    <a href="https://pkg.go.dev/github.com/togo-framework/bot"><img src="https://pkg.go.dev/badge/github.com/togo-framework/bot.svg" alt="pkg.go.dev" /></a>
    <img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT" />
  </p>
  <p><strong>Part of the <a href="https://to-go.dev">togo</a> framework.</strong></p>
</div>

## Install

```bash
togo install togo-framework/bot
togo install togo-framework/bot-telegram   # or bot-discord / bot-slack
```

<!-- /togo-header -->

**bot** is togo's chat-bot subsystem. It defines a small `Bot` driver contract
plus a **command / message handler registry** so your app registers bot commands
*once* and they work across Telegram, Discord and Slack. Real platforms ship as
driver plugins; select one with the `BOT_DRIVER` env var.

| Platform | Plugin | Library |
|----------|--------|---------|
| Telegram | [`bot-telegram`](https://github.com/togo-framework/bot-telegram) | go-telegram-bot-api |
| Discord  | [`bot-discord`](https://github.com/togo-framework/bot-discord)   | discordgo |
| Slack    | [`bot-slack`](https://github.com/togo-framework/bot-slack)       | slack-go (Socket Mode) |

## Usage

Blank-import the base and a driver, then register handlers:

```go
import (
	"context"
	_ "github.com/togo-framework/bot"
	_ "github.com/togo-framework/bot-telegram"
	"github.com/togo-framework/bot"
)

func init() {
	// /weather <city>
	bot.OnCommand("weather", func(ctx context.Context, b *bot.Service, m bot.Message) (string, error) {
		if len(m.Args) == 0 {
			return "usage: /weather <city>", nil
		}
		return "☀️ sunny in " + m.Args[0], nil
	})

	// Catch-all for non-command messages
	bot.OnMessage(func(ctx context.Context, b *bot.Service, m bot.Message) (string, error) {
		return "you said: " + m.Text, nil
	})
}
```

Pick the platform at runtime:

```bash
BOT_DRIVER=telegram
TELEGRAM_BOT_TOKEN=123456:ABC...
```

A built-in `/ping → pong 🏓` command is registered out of the box.

### Sending proactively

From anywhere you have the kernel:

```go
if svc, ok := bot.FromKernel(k); ok {
	_ = svc.Send(ctx, channelID, "deploy finished ✅")
}
```

## Contract

```go
type Bot interface {
	Start(ctx context.Context) error                 // receive loop (blocks)
	Stop() error                                      // shut down
	Send(ctx context.Context, channel, msg string) error
}

type Handler func(ctx context.Context, b *Service, m Message) (string, error)

func RegisterDriver(name string, f DriverFactory) // drivers call this in init()
func OnCommand(name string, h Handler)            // register "/name"
func OnMessage(h Handler)                          // register a catch-all
func FromKernel(k *togo.Kernel) (*Service, bool)  // grab the runtime
```

Writing a new platform driver? Implement `Bot`, call `bot.RegisterDriver` in
`init()`, and invoke the `dispatch` func you're handed for every inbound message.
See `bot-telegram` for the canonical ~120-line example.

## License

MIT © togo-framework

<!-- togo-sponsors -->
---

<div align="center">
  <h3>Premium sponsors</h3>
  <p>
    <a href="https://id8media.com"><strong>ID8 Media</strong></a> &nbsp;·&nbsp;
    <a href="https://one-studio.co"><strong>One Studio</strong></a>
  </p>
  <p><sub>Support togo — <a href="https://github.com/sponsors/fadymondy">become a sponsor</a>.</sub></p>
</div>
<!-- /togo-sponsors -->
