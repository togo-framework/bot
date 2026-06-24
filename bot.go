// Package bot is togo's chat-bot subsystem: a Bot driver contract plus a
// command/message handler registry so any plugin can add bot commands that work
// across platforms. Real platforms (Telegram, Discord, Slack) ship as driver
// plugins that call bot.RegisterDriver and depend on this package. Select one
// with BOT_DRIVER.
//
// Install: `togo install togo-framework/bot` (blank-import registers it), then a
// driver, e.g. `togo install togo-framework/bot-telegram`.
//
//	import _ "github.com/togo-framework/bot"
//	import _ "github.com/togo-framework/bot-telegram"
//
//	bot.OnCommand("ping", func(ctx context.Context, b *bot.Service, m bot.Message) (string, error) {
//		return "pong 🏓", nil
//	})
package bot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/togo-framework/togo"
)

// Message is an inbound chat message normalized across platforms.
type Message struct {
	Channel   string         // platform channel/chat ID — reply here
	User      string         // sender's platform user ID
	Username  string         // sender's display name / handle (best-effort)
	Text      string         // raw message text
	IsCommand bool           // true when Text began with "/"
	Command   string         // command name without the leading "/" (e.g. "ping")
	Args      []string       // whitespace-split arguments after the command
	Platform  string         // driver name: telegram | discord | slack
	Raw       map[string]any // platform-specific payload for advanced handlers
}

// Handler handles a command or message. Return a non-empty string to reply, or
// an error (the error is logged and no reply is sent).
type Handler func(ctx context.Context, b *Service, m Message) (string, error)

// Bot is implemented by platform driver plugins (telegram/discord/slack).
type Bot interface {
	// Start begins receiving messages and blocks until ctx is canceled or Stop
	// is called. Drivers invoke the dispatch func they were given for each message.
	Start(ctx context.Context) error
	// Stop shuts the bot down and releases resources.
	Stop() error
	// Send posts a message to a channel/chat.
	Send(ctx context.Context, channel, msg string) error
}

// DriverFactory builds a Bot from the kernel (env-configured). dispatch must be
// called by the driver for every inbound message; it routes to the registered
// command/message handlers and sends any reply.
type DriverFactory func(k *togo.Kernel, dispatch func(context.Context, Message)) (Bot, error)

var (
	regMu       sync.RWMutex
	drivers     = map[string]DriverFactory{}
	cmdMu       sync.RWMutex
	commands    = map[string]Handler{}
	msgMu       sync.RWMutex
	msgHandlers []Handler
)

// RegisterDriver registers a bot platform driver by name (call from a plugin's
// init()).
func RegisterDriver(name string, f DriverFactory) {
	regMu.Lock()
	drivers[name] = f
	regMu.Unlock()
}

// Drivers lists the registered driver names (for diagnostics).
func Drivers() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]string, 0, len(drivers))
	for n := range drivers {
		out = append(out, n)
	}
	return out
}

// OnCommand registers a handler for "/name". A leading slash is optional. The
// handler runs when a message starts with "/name". Later registrations for the
// same name win.
func OnCommand(name string, h Handler) {
	cmdMu.Lock()
	commands[strings.TrimPrefix(name, "/")] = h
	cmdMu.Unlock()
}

// OnMessage registers a handler invoked for every non-command message.
func OnMessage(h Handler) {
	msgMu.Lock()
	msgHandlers = append(msgHandlers, h)
	msgMu.Unlock()
}

// Service is the bot runtime stored on the kernel (k.Get("bot")).
type Service struct {
	bot    Bot
	driver string
	k      *togo.Kernel
}

// Bot returns the active platform driver.
func (s *Service) Bot() Bot { return s.bot }

// log returns the kernel logger, falling back to slog's default.
func (s *Service) log() *slog.Logger {
	if s.k != nil && s.k.Log != nil {
		return s.k.Log
	}
	return slog.Default()
}

// Driver returns the active driver name (e.g. "telegram").
func (s *Service) Driver() string { return s.driver }

// Send posts a message to a channel using the active driver.
func (s *Service) Send(ctx context.Context, channel, msg string) error {
	if s.bot == nil {
		return fmt.Errorf("bot: no active driver")
	}
	return s.bot.Send(ctx, channel, msg)
}

// ParseCommand splits "/cmd a b c" into ("cmd", ["a","b","c"], true). It returns
// ok=false when text is not a command.
func ParseCommand(text string) (cmd string, args []string, ok bool) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", nil, false
	}
	fields := strings.Fields(text[1:])
	if len(fields) == 0 {
		return "", nil, false
	}
	// Telegram commands can be "/cmd@BotName" — strip the @suffix.
	cmd = fields[0]
	if at := strings.IndexByte(cmd, '@'); at >= 0 {
		cmd = cmd[:at]
	}
	return cmd, fields[1:], true
}

// Dispatch routes m to the matching command handler (if any) or to the message
// handlers, and sends back any non-empty reply. Drivers call this for each
// inbound message.
func (s *Service) Dispatch(ctx context.Context, m Message) {
	if cmd, args, ok := ParseCommand(m.Text); ok {
		m.IsCommand = true
		m.Command = cmd
		m.Args = args
		cmdMu.RLock()
		h, found := commands[cmd]
		cmdMu.RUnlock()
		if found {
			s.run(ctx, h, m)
			return
		}
		// Unknown command: still fall through to message handlers.
	}
	msgMu.RLock()
	hs := append([]Handler(nil), msgHandlers...)
	msgMu.RUnlock()
	for _, h := range hs {
		s.run(ctx, h, m)
	}
}

func (s *Service) run(ctx context.Context, h Handler, m Message) {
	reply, err := h(ctx, s, m)
	if err != nil {
		s.log().Error("bot: handler error", "err", err, "platform", m.Platform, "command", m.Command)
		return
	}
	if reply != "" {
		if err := s.Send(ctx, m.Channel, reply); err != nil {
			s.log().Error("bot: send reply failed", "err", err, "platform", m.Platform)
		}
	}
}

func init() {
	// Built-in demo command so a fresh bot answers something out of the box.
	OnCommand("ping", func(ctx context.Context, b *Service, m Message) (string, error) {
		return "pong 🏓", nil
	})

	togo.RegisterProviderFunc("bot", togo.PriorityLate, func(k *togo.Kernel) error {
		name := os.Getenv("BOT_DRIVER")
		if name == "" {
			// No bot configured — register nothing (no-op). Handlers can still be
			// added; they simply won't receive messages until a driver is selected.
			return nil
		}
		regMu.RLock()
		f, ok := drivers[name]
		regMu.RUnlock()
		if !ok {
			return fmt.Errorf("bot: unknown driver %q (install its plugin, e.g. togo install togo-framework/bot-%s)", name, name)
		}
		svc := &Service{driver: name, k: k}
		b, err := f(k, svc.Dispatch)
		if err != nil {
			return err
		}
		svc.bot = b
		k.Set("bot", svc)

		// Run the receive loop in the background so it doesn't block boot.
		go func() {
			if err := b.Start(context.Background()); err != nil {
				k.Log.Error("bot: driver stopped", "err", err, "driver", name)
			}
		}()
		return nil
	})
}

// FromKernel returns the bot Service if a driver was registered.
func FromKernel(k *togo.Kernel) (*Service, bool) {
	v, ok := k.Get("bot")
	if !ok {
		return nil, false
	}
	s, ok := v.(*Service)
	return s, ok
}
