package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestHandleStatusClear(t *testing.T) {
	t.Run("matching id clears status", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.StatusMsg = "Value updated"
		m.statusID = 3
		result, cmd := m.handleStatusClear(types.StatusClearMsg{ID: 3})
		if cmd != nil {
			t.Fatal("expected nil cmd")
		}
		if result.(Model).StatusMsg != "" {
			t.Errorf("StatusMsg = %q, want empty", result.(Model).StatusMsg)
		}
	})
	t.Run("stale id leaves status", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.StatusMsg = "Value updated"
		m.statusID = 4
		result, cmd := m.handleStatusClear(types.StatusClearMsg{ID: 3})
		if cmd != nil {
			t.Fatal("expected nil cmd")
		}
		if result.(Model).StatusMsg != "Value updated" {
			t.Errorf("StatusMsg = %q, want %q", result.(Model).StatusMsg, "Value updated")
		}
	})
}

func TestScheduleStatusClear(t *testing.T) {
	t.Run("unchanged status is a no-op", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.StatusMsg = "Value updated"
		m.statusID = 2
		if cmd := m.scheduleStatusClear("Value updated"); cmd != nil {
			t.Fatal("expected nil cmd")
		}
		if m.statusID != 2 {
			t.Errorf("statusID = %d, want 2", m.statusID)
		}
	})
	t.Run("cleared status invalidates pending toast", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.StatusMsg = ""
		m.statusID = 2
		if cmd := m.scheduleStatusClear("Value updated"); cmd != nil {
			t.Fatal("expected nil cmd")
		}
		if m.statusID != 3 {
			t.Errorf("statusID = %d, want 3", m.statusID)
		}
	})
	t.Run("sticky status is not timed out", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.StatusMsg = "Connecting..."
		if cmd := m.scheduleStatusClear(""); cmd != nil {
			t.Fatal("expected nil cmd for sticky status")
		}
		if m.statusID != 1 {
			t.Errorf("statusID = %d, want 1", m.statusID)
		}
	})
	t.Run("transient status schedules clear", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.StatusMsg = "Value updated"
		cmd := m.scheduleStatusClear("")
		if cmd == nil {
			t.Fatal("expected clear cmd")
		}
		if m.statusID != 1 {
			t.Errorf("statusID = %d, want 1", m.statusID)
		}
	})
}

func TestStatusClearAfter(t *testing.T) {
	cmd := statusClearAfter(7, time.Millisecond)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	clear, ok := msg.(types.StatusClearMsg)
	if !ok {
		t.Fatalf("expected StatusClearMsg, got %T", msg)
	}
	if clear.ID != 7 {
		t.Errorf("ID = %d, want 7", clear.ID)
	}
}

func TestStatusDuration(t *testing.T) {
	if got := statusDuration("Value updated"); got != statusFlashDuration {
		t.Errorf("success duration = %v, want %v", got, statusFlashDuration)
	}
	if got := statusDuration("Error: boom"); got != statusErrorDuration {
		t.Errorf("error duration = %v, want %v", got, statusErrorDuration)
	}
}

func TestIsStickyStatus(t *testing.T) {
	if !isStickyStatus("Connecting...") {
		t.Error("Connecting... should be sticky")
	}
	if !isStickyStatus("Watching key for changes...") {
		t.Error("Watching key for changes... should be sticky")
	}
	if isStickyStatus("Value updated") {
		t.Error("Value updated should not be sticky")
	}
}

func TestIsErrorStatus(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"Error: boom", true},
		{"Copy failed: x", true},
		{"Batch TTL error: x", true},
		{"Invalid TTL: must be an integer (seconds)", true},
		{"Value updated", false},
		{"Connected", false},
	}
	for _, tc := range cases {
		if got := isErrorStatus(tc.msg); got != tc.want {
			t.Errorf("isErrorStatus(%q) = %v, want %v", tc.msg, got, tc.want)
		}
	}
}

func TestUpdate_StatusClearMsg(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.StatusMsg = "Value updated"
	m.statusID = 1
	result, cmd := m.Update(types.StatusClearMsg{ID: 1})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	if result.(Model).StatusMsg != "" {
		t.Errorf("StatusMsg = %q, want empty", result.(Model).StatusMsg)
	}
}

func TestUpdate_SchedulesStatusClear(t *testing.T) {
	t.Run("success toast", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, cmd := m.Update(types.ClipboardCopiedMsg{})
		model := result.(Model)
		if model.StatusMsg != "Copied to clipboard" {
			t.Fatalf("StatusMsg = %q, want Copied to clipboard", model.StatusMsg)
		}
		if cmd == nil {
			t.Fatal("expected status clear cmd")
		}
		if model.statusID == 0 {
			t.Error("expected statusID to increment")
		}
	})
	t.Run("sticky connecting does not schedule", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, cmd := m.Update(types.AutoConnectMsg{Connection: types.Connection{Name: "local"}})
		model := result.(Model)
		if model.StatusMsg != "Connecting..." {
			t.Fatalf("StatusMsg = %q, want Connecting...", model.StatusMsg)
		}
		if cmd == nil {
			t.Fatal("expected connect cmd")
		}
		msg := cmd()
		if _, ok := msg.(types.StatusClearMsg); ok {
			t.Fatal("sticky status must not emit StatusClearMsg")
		}
	})
	t.Run("unchanged status does not reschedule", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.StatusMsg = "Value updated"
		m.statusID = 9
		result, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		if cmd != nil {
			t.Fatal("expected nil cmd when status is unchanged")
		}
		if result.(Model).statusID != 9 {
			t.Errorf("statusID = %d, want 9", result.(Model).statusID)
		}
	})
}
