package cmd

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestPublishMessage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.PublishResult = 2
		msg := cmds.PublishMessage("chan", "hello")()
		result := msg.(types.PublishResultMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Receivers != 2 {
			t.Errorf("Receivers = %d, want 2", result.Receivers)
		}
		if result.Channel != "chan" {
			t.Errorf("Channel = %q, want %q", result.Channel, "chan")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.PublishMessage("ch", "msg")()
		result := msg.(types.PublishResultMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestGetPubSubChannels(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.PubSubChannelsResult = []string{"chan1", "chan2"}
		msg := cmds.GetPubSubChannels("*")()
		result := msg.(types.PubSubChannelsLoadedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if len(result.Channels) != 2 {
			t.Errorf("expected 2 channels, got %d", len(result.Channels))
		}
		if result.Channels[0].Name != "chan1" {
			t.Errorf("Channels[0].Name = %q, want %q", result.Channels[0].Name, "chan1")
		}
		if result.Channels[1].Name != "chan2" {
			t.Errorf("Channels[1].Name = %q, want %q", result.Channels[1].Name, "chan2")
		}
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.PubSubChannelsError = errors.New("pubsub error")
		msg := cmds.GetPubSubChannels("*")()
		result := msg.(types.PubSubChannelsLoadedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.GetPubSubChannels("*")()
		result := msg.(types.PubSubChannelsLoadedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestSubscribeKeyspace(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, _ := newMockCmds()
		var received []tea.Msg
		sendFunc := func(msg tea.Msg) {
			received = append(received, msg)
		}
		msg := cmds.SubscribeKeyspace("*", sendFunc)()
		result := msg.(types.KeyspaceSubscribedMsg)
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Pattern != "*" {
			t.Errorf("Pattern = %q, want %q", result.Pattern, "*")
		}
	})

	t.Run("forwards events via sendFunc", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.SubscribeKeyspaceEvents = []types.KeyspaceEvent{
			{Key: "k1", Event: "set"},
			{Key: "k2", Event: "del"},
		}
		var received []tea.Msg
		sendFunc := func(msg tea.Msg) { received = append(received, msg) }
		_ = cmds.SubscribeKeyspace("*", sendFunc)()
		if len(received) != 2 {
			t.Fatalf("expected 2 forwarded events, got %d", len(received))
		}
		ev, ok := received[0].(types.KeyspaceEventMsg)
		if !ok {
			t.Fatalf("expected KeyspaceEventMsg, got %T", received[0])
		}
		if ev.Event.Key != "k1" {
			t.Errorf("Key = %q, want %q", ev.Event.Key, "k1")
		}
	})

	t.Run("nil sendFunc is tolerated", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.SubscribeKeyspaceEvents = []types.KeyspaceEvent{{Key: "k", Event: "set"}}
		_ = cmds.SubscribeKeyspace("*", nil)()
	})

	t.Run("error", func(t *testing.T) {
		cmds, mock := newMockCmds()
		mock.SubscribeKeyspaceError = errors.New("subscribe failed")
		msg := cmds.SubscribeKeyspace("*", nil)()
		result := msg.(types.KeyspaceSubscribedMsg)
		if result.Err == nil {
			t.Error("expected error")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.SubscribeKeyspace("*", nil)()
		result := msg.(types.KeyspaceSubscribedMsg)
		if result.Err != nil {
			t.Errorf("nil redis should not error: %v", result.Err)
		}
	})
}

func TestUnsubscribeKeyspace(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmds, mock := newMockCmds()
		msg := cmds.UnsubscribeKeyspace()()
		if msg != nil {
			t.Errorf("expected nil msg, got %T", msg)
		}
		found := false
		for _, call := range mock.Calls {
			if call == "UnsubscribeKeyspace" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected UnsubscribeKeyspace to be called")
		}
	})

	t.Run("nil redis", func(t *testing.T) {
		cmds := NewCommands(nil, nil)
		msg := cmds.UnsubscribeKeyspace()()
		if msg != nil {
			t.Errorf("expected nil msg, got %T", msg)
		}
	})
}
