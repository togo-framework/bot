package bot

import (
	"context"
	"sync"
	"testing"
)

func TestParseCommand(t *testing.T) {
	cases := []struct {
		in       string
		wantCmd  string
		wantArgs []string
		wantOK   bool
	}{
		{"/ping", "ping", []string{}, true},
		{"  /echo hello world ", "echo", []string{"hello", "world"}, true},
		{"/start@MyTogoBot now", "start", []string{"now"}, true},
		{"hello", "", nil, false},
		{"/", "", nil, false},
		{"", "", nil, false},
	}
	for _, c := range cases {
		cmd, args, ok := ParseCommand(c.in)
		if ok != c.wantOK || cmd != c.wantCmd {
			t.Errorf("ParseCommand(%q) = (%q,%v,%v), want (%q,_,%v)", c.in, cmd, args, ok, c.wantCmd, c.wantOK)
		}
		if c.wantOK && len(args) != len(c.wantArgs) {
			t.Errorf("ParseCommand(%q) args = %v, want %v", c.in, args, c.wantArgs)
		}
	}
}

// fakeBot records sent messages so we can assert on replies.
type fakeBot struct {
	mu   sync.Mutex
	sent []struct{ ch, msg string }
}

func (f *fakeBot) Start(ctx context.Context) error { return nil }
func (f *fakeBot) Stop() error                      { return nil }
func (f *fakeBot) Send(ctx context.Context, ch, msg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, struct{ ch, msg string }{ch, msg})
	return nil
}

func TestDispatchCommand(t *testing.T) {
	fb := &fakeBot{}
	svc := &Service{bot: fb, driver: "fake"}

	got := ""
	OnCommand("greet", func(ctx context.Context, b *Service, m Message) (string, error) {
		got = m.Command
		if len(m.Args) > 0 {
			return "hi " + m.Args[0], nil
		}
		return "hi", nil
	})

	svc.Dispatch(context.Background(), Message{Channel: "c1", Text: "/greet sam"})

	if got != "greet" {
		t.Fatalf("handler command = %q, want greet", got)
	}
	fb.mu.Lock()
	defer fb.mu.Unlock()
	if len(fb.sent) != 1 || fb.sent[0].msg != "hi sam" {
		t.Fatalf("sent = %+v, want one reply 'hi sam'", fb.sent)
	}
}

func TestDispatchMessage(t *testing.T) {
	fb := &fakeBot{}
	svc := &Service{bot: fb, driver: "fake"}

	OnMessage(func(ctx context.Context, b *Service, m Message) (string, error) {
		if m.Text == "knock" {
			return "who's there", nil
		}
		return "", nil
	})

	svc.Dispatch(context.Background(), Message{Channel: "c2", Text: "knock"})

	fb.mu.Lock()
	defer fb.mu.Unlock()
	found := false
	for _, s := range fb.sent {
		if s.msg == "who's there" {
			found = true
		}
	}
	if !found {
		t.Fatalf("message handler reply not sent: %+v", fb.sent)
	}
}

func TestBuiltinPing(t *testing.T) {
	cmdMu.RLock()
	h, ok := commands["ping"]
	cmdMu.RUnlock()
	if !ok {
		t.Fatal("builtin ping command not registered")
	}
	reply, err := h(context.Background(), &Service{}, Message{Text: "/ping"})
	if err != nil || reply == "" {
		t.Fatalf("ping = (%q,%v), want non-empty reply", reply, err)
	}
}
