package editor

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestNew(t *testing.T) {
	m := New("hello", 80, 24, "file.json")
	if m == nil {
		t.Fatal("expected non-nil editor")
	}
	if m.Value() != "hello" {
		t.Errorf("expected content hello, got %q", m.Value())
	}
	if m.FileName() != "file.json" {
		t.Errorf("expected fileName file.json, got %q", m.FileName())
	}
}

func TestSetSize(t *testing.T) {
	m := New("content", 40, 10, "")
	m.SetSize(120, 30)
	if m.View() == "" {
		t.Error("expected non-empty view after resize")
	}
}

func TestView(t *testing.T) {
	m := New("line1\nline2", 80, 24, "")
	if m.View() == "" {
		t.Error("expected non-empty view")
	}
}

func TestSave(t *testing.T) {
	m := New("payload", 80, 24, "")
	cmd := m.Save()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg, ok := cmd().(types.EditorSaveMsg)
	if !ok {
		t.Fatalf("expected EditorSaveMsg, got %T", cmd())
	}
	if msg.Content != "payload" {
		t.Errorf("expected content payload, got %q", msg.Content)
	}
}

func TestCancel(t *testing.T) {
	m := New("payload", 80, 24, "")
	cmd := m.Cancel()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	if _, ok := cmd().(types.EditorQuitMsg); !ok {
		t.Fatalf("expected EditorQuitMsg, got %T", cmd())
	}
}

func TestUpdate(t *testing.T) {
	t.Run("ctrl+s saves", func(t *testing.T) {
		m := New("data", 80, 24, "")
		updated, cmd := m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
		if updated == nil {
			t.Fatal("expected non-nil editor")
		}
		if cmd == nil {
			t.Fatal("expected save cmd")
		}
		msg, ok := cmd().(types.EditorSaveMsg)
		if !ok {
			t.Fatalf("expected EditorSaveMsg, got %T", cmd())
		}
		if msg.Content != "data" {
			t.Errorf("expected content data, got %q", msg.Content)
		}
	})

	t.Run("esc cancels", func(t *testing.T) {
		m := New("data", 80, 24, "")
		_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
		if cmd == nil {
			t.Fatal("expected cancel cmd")
		}
		if _, ok := cmd().(types.EditorQuitMsg); !ok {
			t.Fatalf("expected EditorQuitMsg, got %T", cmd())
		}
	})

	t.Run("ctrl+q cancels", func(t *testing.T) {
		m := New("data", 80, 24, "")
		_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
		if cmd == nil {
			t.Fatal("expected cancel cmd")
		}
		if _, ok := cmd().(types.EditorQuitMsg); !ok {
			t.Fatalf("expected EditorQuitMsg, got %T", cmd())
		}
	})

	t.Run("text key edits content", func(t *testing.T) {
		m := New("", 80, 24, "")
		updated, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
		if updated.Value() != "a" {
			t.Errorf("expected content a, got %q", updated.Value())
		}
	})

	t.Run("non-key message delegates", func(t *testing.T) {
		m := New("data", 80, 24, "")
		updated, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
		if updated == nil {
			t.Fatal("expected non-nil editor")
		}
		_ = cmd
	})
}
