package types

import "testing"

func TestDefaultKeyBindings(t *testing.T) {
	kb := DefaultKeyBindings()

	t.Run("critical bindings non-empty", func(t *testing.T) {
		checks := []struct {
			name  string
			value string
		}{
			{"Quit", kb.Quit},
			{"Select", kb.Select},
			{"Back", kb.Back},
			{"Help", kb.Help},
			{"Up", kb.Up},
			{"Down", kb.Down},
			{"Delete", kb.Delete},
			{"Search", kb.Search},
			{"Refresh", kb.Refresh},
		}
		for _, c := range checks {
			if c.value == "" {
				t.Errorf("DefaultKeyBindings().%s is empty", c.name)
			}
		}
	})
}

func TestGetBindingsList(t *testing.T) {
	kb := DefaultKeyBindings()
	bindings := kb.GetBindingsList()

	t.Run("correct count", func(t *testing.T) {
		if len(bindings) != 40 {
			t.Errorf("GetBindingsList() returned %d entries, want 40", len(bindings))
		}
	})

	t.Run("each has non-empty fields", func(t *testing.T) {
		for i, b := range bindings {
			if b.Key == "" {
				t.Errorf("binding[%d].Key is empty (Action=%q)", i, b.Action)
			}
			if b.Description == "" {
				t.Errorf("binding[%d].Description is empty (Action=%q)", i, b.Action)
			}
			if b.Action == "" {
				t.Errorf("binding[%d].Action is empty (Key=%q)", i, b.Key)
			}
		}
	})
}

func TestScreenString(t *testing.T) {
	tests := []struct {
		name     string
		screen   Screen
		expected string
	}{
		{"connections screen", ScreenConnections, "Connections"},
		{"keys screen", ScreenKeys, "Keys"},
		{"help screen", ScreenHelp, "Help"},
		{"server info screen", ScreenServerInfo, "Server Info"},
		{"live metrics screen", ScreenLiveMetrics, "Live Metrics"},
		{"unknown screen", Screen(9999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.screen.String()
			if got != tt.expected {
				t.Errorf("Screen(%d).String() = %q, want %q", tt.screen, got, tt.expected)
			}
		})
	}
}
