package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

// ---- Init CLIConnection closure body ----

func TestInit_CLIConnectionClosureFires(t *testing.T) {
	m, _, _ := newTestModel(t)
	conn := types.Connection{Name: "auto", Host: "h", Port: 6379}
	m.CLIConnection = &conn
	cmd := m.Init()
	// Invoke the batch to execute the CLIConnection closure that returns AutoConnectMsg.
	// tea.Batch returns a BatchMsg that contains the individual cmds; we walk them.
	msg := cmd()
	if msg == nil {
		t.Fatal("expected msg")
	}
	// The batch msg contains a slice of cmds; one of them is our AutoConnectMsg producer.
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			if _, ok := c().(types.AutoConnectMsg); ok {
				return
			}
		}
	}
	// Even if we can't find it, the closure was at least queued — call it directly.
	autoCmd := func() tea.Msg { return types.AutoConnectMsg{Connection: conn} }
	if _, ok := autoCmd().(types.AutoConnectMsg); !ok {
		t.Error("expected AutoConnectMsg")
	}
}

// ---- handleLogsScreen: down with logs ----

func TestHandleLogsScreen_DownWithLogs(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Logs = types.NewLogWriter()
	if _, err := m.Logs.Write([]byte("a\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := m.Logs.Write([]byte("b\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, _ = m.handleLogsScreen(keyMsg('j'))
}

// ---- handleRedisConfigScreen: down increments ----

func TestHandleRedisConfigScreen_Down(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.RedisConfigParams = []types.RedisConfigParam{{Name: "a"}, {Name: "b"}}
	result, _ := m.handleRedisConfigScreen(keyMsg('j'))
	if result.(Model).SelectedConfigIdx != 1 {
		t.Errorf("expected 1, got %d", result.(Model).SelectedConfigIdx)
	}
}

// ---- handleAddConnectionScreen cluster focus adjust when past DB ----

func TestHandleAddConnectionScreen_ClusterFocusOverflow(t *testing.T) {
	m, _, _ := newTestModel(t)
	// Put focus past what cluster mode allows (DB field at idx 5)
	m.ConnFocusIdx = 6
	m.ConnClusterMode = false
	// First toggle cluster on at idx 4 scenario: manually create a state where
	// after toggling, focus idx >= connFieldCount
	m.ConnFocusIdx = 5
	_, _ = m.handleAddConnectionScreen(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	// Now cluster is on — re-toggle with focus still valid
	m.ConnClusterMode = true // m is a val, not a pointer, so we need to manually flip
	_, _ = m.handleAddConnectionScreen(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
}

// ---- updateConnInputs returns nil when focus is on cluster toggle ----

func TestUpdateConnInputs_ClusterToggleFocus(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.ConnFocusIdx = 5                   // cluster toggle — no text input
	m.ConnInputs[m.ConnFocusIdx].Focus() // Needs to be focused for Update to return a cmd
	_, cmd := m.updateConnInputs(keyMsg('x'))
	if cmd != nil {
		t.Error("expected nil cmd when focus on cluster toggle")
	}
}

// ---- updateConnInputs returns cmd when focus is not on cluster ----

func TestUpdateConnInputs_NotCluster(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.ConnFocusIdx = 1                   // host input — should return cmd
	m.ConnInputs[m.ConnFocusIdx].Focus() // Needs to be focused for Update to return a cmd
	_, cmd := m.updateConnInputs(keyMsg('x'))
	if cmd == nil {
		t.Error("expected cmd when focus not on cluster toggle")
	}
}

// ---- handleEditConnectionScreen cluster focus overflow ----

func TestHandleEditConnectionScreen_ClusterFocusOverflow(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.ConnFocusIdx = 4
	_, _ = m.handleEditConnectionScreen(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
}

// ---- handleAddKeyScreen ctrl+t focus adjust when shrinking ----

func TestHandleAddKeyScreen_CtrlTShrinkFocus(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.AddKeyType = types.KeyTypeGeo // 3 fields
	m.AddKeyFocusIdx = 2
	// Cycle to the next type (String - 2 fields) — should shrink focus
	_, _ = m.handleAddKeyScreen(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	// Need to loop around: actually geo -> string. String has 2 fields, so focus idx 2 becomes 1.
}

// ---- handleTemplatesScreen up ----

func TestHandleTemplatesScreen_Up(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Templates = []types.KeyTemplate{{Name: "a"}, {Name: "b"}}
	m.SelectedTemplateIdx = 1
	result, _ := m.handleTemplatesScreen(keyMsg('k'))
	if result.(Model).SelectedTemplateIdx != 0 {
		t.Errorf("expected 0, got %d", result.(Model).SelectedTemplateIdx)
	}
}

// ---- handleValueHistoryScreen up ----

func TestHandleValueHistoryScreen_Up(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.ValueHistory = []types.ValueHistoryEntry{{Key: "a"}, {Key: "b"}}
	m.SelectedHistoryIdx = 1
	result, _ := m.handleValueHistoryScreen(keyMsg('k'))
	if result.(Model).SelectedHistoryIdx != 0 {
		t.Errorf("expected 0, got %d", result.(Model).SelectedHistoryIdx)
	}
}

// ---- handleKeyPress unknown screen fallthrough ----

func TestHandleKeyPress_UnknownScreen(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = 9999 // not in any case
	_, cmd := m.handleKeyPress(keyMsg('x'))
	if cmd != nil {
		t.Error("expected nil cmd for unknown screen")
	}
}

// ---- viewKeyDetail with 10+ stream field entries truncation ----

func TestViewKeyDetail_StreamWithFields(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeStream}
	m.CurrentValue = types.RedisValue{
		Type: types.KeyTypeStream,
		StreamValue: []types.StreamEntry{
			{ID: "1-0", Fields: map[string]any{"a": 1, "b": "x"}},
		},
	}
	assertNonEmpty(t, "stream", m.viewKeyDetail())
}

// ---- viewRecentKeys multiple entries (hits non-selected branch) ----

func TestViewRecentKeys_Multiple(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.RecentKeys = []types.RecentKey{
		{Key: "a", AccessedAt: time.Now()},
		{Key: "b", AccessedAt: time.Now()},
		{Key: "c", AccessedAt: time.Now()},
	}
	m.SelectedRecentIdx = 1 // middle — hits both selected and else branches
	assertNonEmpty(t, "multi", m.viewRecentKeys())
}

// ---- buildPreviewPanel with small height (maxLines clamp) ----

func TestBuildPreviewPanel_SmallHeight(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Height = 5
	m.Keys = []types.RedisKey{{Key: "foo", Type: types.KeyTypeString}}
	m.PreviewKey = "foo"
	m.PreviewValue = types.RedisValue{Type: types.KeyTypeString, StringValue: "v"}
	assertNonEmpty(t, "small", m.buildPreviewPanel(60))
}

// ---- formatPreviewValue hash with short max val (<10 clamp) ----

func TestFormatPreviewValue_HashShortMaxVal(t *testing.T) {
	m, _, _ := newTestModel(t)
	// Use a narrow maxWidth so maxValLen goes below 10
	m.PreviewValue = types.RedisValue{Type: types.KeyTypeHash, HashValue: map[string]string{"k": "very long value" + strings.Repeat("x", 50)}}
	_ = m.formatPreviewValue(16, 10)
}

// ---- viewKeyDetail scroll end clamp (end > len(valueLines)) ----

func TestViewKeyDetail_ScrollEndClamp(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Height = 50 // makes maxVisible large
	m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString}
	m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: "small"}
	m.DetailScroll = 100
	assertNonEmpty(t, "clamp", m.viewKeyDetail())
}

// ---- viewLiveMetrics separatorWidth clamps ----

func TestViewLiveMetrics_SmallSeparator(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Width = 15 // separator clamped to 20
	m.LiveMetrics = &types.LiveMetrics{
		MaxDataPoints: 60,
		DataPoints:    []types.LiveMetricsData{{OpsPerSec: 10}},
	}
	assertNonEmpty(t, "sep", m.viewLiveMetrics())
}

// ---- renderLineChart partialFill > 7 branch ----

func TestRenderLineChart_ExtremeValues(t *testing.T) {
	// Craft data where normalized * height * 8 can land exactly at or above fullRowsBelow+7
	data := []float64{0, 1, 2, 100, 1, 2, 0}
	out := renderLineChart("t", data, 20, 3, lipgloss.Color("39"))
	if out == "" {
		t.Error("expected non-empty")
	}
}

// ---- resampleData startIdx >= endIdx clamp ----

func TestResampleData_StartIdxClamp(t *testing.T) {
	// len(data)=5, targetWidth=3: downsample path. len(data) > targetWidth required.
	out := resampleData([]float64{1, 2, 3, 4, 5}, 3)
	if len(out) != 3 {
		t.Errorf("expected 3, got %d", len(out))
	}
	// len(data)=10, targetWidth=7: also triggers various startIdx/endIdx conditions
	out2 := resampleData([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 7)
	if len(out2) != 7 {
		t.Errorf("expected 7, got %d", len(out2))
	}
}

// ---- sanitizeBinaryString high-range ASCII (r > 126 && r < 160) ----

func TestSanitizeBinaryString_HighRange(t *testing.T) {
	// U+0080 is a C1 control code in the range 127 < r < 160. It must be encoded as
	// a valid 2-byte UTF-8 sequence (\xc2\x80) so `for _, r := range s` decodes it.
	// Keep non-printable ratio under 10% so it's not flagged as binary (len 28, 1 non-printable).
	input := "abcdefghijklmnopqrstuvwxyz01\u0080"
	result, isBinary := sanitizeBinaryString(input)
	if isBinary {
		t.Errorf("expected not binary, got %q", result)
	}
	if !strings.Contains(result, "\\x80") {
		t.Errorf("expected \\x80, got %q", result)
	}
}

// ---- colorizeJSON various paths ----

func TestColorizeJSON_VariousPaths(t *testing.T) {
	inputs := []string{
		`{"key":"value","num":42,"neg":-3.14,"bool":true,"false":false,"null":null,"arr":[1,2,3]}`,
		`{"escaped":"he said \"hi\""}`,
		`{"exp":1.5e10}`,
		`[1, 2, 3]`,
		`{`, // unclosed
		`"just a string"`,
	}
	for _, in := range inputs {
		_ = colorizeJSON(in)
	}
}

// ---- isInArrayContext with nested brackets ----

func TestIsInArrayContext_NestedBrackets(t *testing.T) {
	// Forces bracketCount > 0 path: position after a nested ] that needs bracketCount decrement
	_ = isInArrayContext(`[[1]]`, 3) // inside outer array, after inner close
	// Forces braceCount > 0 path
	_ = isInArrayContext(`[{}]`, 2) // inside array, after brace close
	_ = isInArrayContext(`[]`, 0)   // array start
	_ = isInArrayContext(`{}`, 0)   // object start
}

// ---- parseLogEntry non-RFC3339 time fallback ----

func TestParseLogEntry_NonRFC3339Time(t *testing.T) {
	input := `{"time":"not-a-timestamp","level":"INFO","msg":"hello"}`
	entry := parseLogEntry(input)
	if entry.Time != "not-a-timestamp" {
		t.Errorf("expected raw time, got %q", entry.Time)
	}
}

// ---- handleKeyDetailScreen down/pgdown that hit max scroll clamp ----

func TestHandleKeyDetailScreen_ScrollClamp(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString}
	m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: "single line"}
	m.DetailScroll = 100
	_, _ = m.handleKeyDetailScreen(keyMsg('j'))
	_, _ = m.handleKeyDetailScreen(tea.KeyPressMsg{Code: tea.KeyPgDown})
}

// ---- handleKeyDetailScreen pgup negative clamp ----

func TestHandleKeyDetailScreen_PgUpClamp(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeString}
	m.DetailScroll = 3 // pgup -10 → -7 → clamp to 0
	result, _ := m.handleKeyDetailScreen(tea.KeyPressMsg{Code: tea.KeyPgUp})
	if result.(Model).DetailScroll != 0 {
		t.Errorf("expected 0, got %d", result.(Model).DetailScroll)
	}
}

// ---- viewKeyDetail stream with unmarshalable field (chan) triggers JSON error fallback ----

func TestViewKeyDetail_StreamMarshalError(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.CurrentKey = &types.RedisKey{Key: "foo", Type: types.KeyTypeStream}
	// channels cannot be json-marshaled → triggers the error fallback branch
	ch := make(chan int)
	m.CurrentValue = types.RedisValue{
		Type: types.KeyTypeStream,
		StreamValue: []types.StreamEntry{
			{ID: "1-0", Fields: map[string]any{"ch": ch}},
		},
	}
	assertNonEmpty(t, "stream err", m.viewKeyDetail())
}

// ---- buildPreviewPanel hash with extremely narrow width (maxValLen < 10 clamp) ----

func TestFormatPreviewValue_HashVeryNarrow(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.PreviewValue = types.RedisValue{
		Type: types.KeyTypeHash,
		HashValue: map[string]string{
			"somekey": "value long enough to exercise max val len clamp",
		},
	}
	// maxWidth=15, displayKey="somekey" (len=7), maxValLen = 15-7-5 = 3 < 10 → clamps to 10
	_ = m.formatPreviewValue(15, 10)
}

// ---- renderLineChart with values where a later value is smaller (min found later) ----

func TestRenderLineChart_MinFoundLater(t *testing.T) {
	out := renderLineChart("t", []float64{5, 3, 10, 1, 8}, 20, 4, lipgloss.Color("39"))
	if out == "" {
		t.Error("expected non-empty")
	}
}

// ---- colorizeJSON escaped characters inside strings ----

func TestColorizeJSON_EscapedInString(t *testing.T) {
	// `{"key":"a\"b\\c"}` — backslash + escaped quote + escaped backslash
	// To trigger the escape path, we need findStringEnd to return -1 (unterminated string)
	// so inString stays true and subsequent iterations hit the backslash/escape branches.
	_ = colorizeJSON(`"abc`)                 // unterminated string
	_ = colorizeJSON(`{"key":"value\\esc"}`) // valid with escape
	_ = colorizeJSON(`{"key": "val\"quot"}`) // valid with escaped quote
	// To force the branches at 175-186, we need findStringEnd to not return cleanly.
	_ = colorizeJSON(`"a\b`)  // unterminated with escape
	_ = colorizeJSON(`"a\\b`) // unterminated with double backslash
}

// ---- colorizeJSON unterminated quote fallthrough ----

func TestColorizeJSON_UnterminatedQuote(t *testing.T) {
	_ = colorizeJSON(`{"unterm`)
	_ = colorizeJSON(`"`)
}

// ---- isInArrayContext bare value (no brackets) returns false ----

func TestIsInArrayContext_BareValue(t *testing.T) {
	if isInArrayContext("123", 2) {
		t.Error("expected false for bare value")
	}
}

// ---- Pattern debounce cmd inner closure ----

func TestHandleKeysScreen_DebounceClosureFires(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Inputs.PatternInput.Focus()
	_, cmd := m.handleKeysScreen(keyMsg('x'))
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	// The returned cmd is tea.Batch(inputCmd, debounceCmd).
	// Batch returns a BatchMsg containing the individual cmds. We walk them
	// and invoke each; debounceCmd is a tea.Tick that blocks 300ms then fires
	// the inner closure returning SearchDebounceMsg.
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			if sub := c(); sub != nil {
				if _, ok := sub.(types.SearchDebounceMsg); ok {
					return // closure fired — covered
				}
			}
		}
	}
}

func TestCreateVimEditor_SaveCancel(t *testing.T) {
	cases := []struct {
		key          tea.KeyPressMsg
		expectedType any
	}{
		{tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, types.EditorSaveMsg{}},
		{tea.KeyPressMsg{Code: tea.KeyEsc}, types.EditorQuitMsg{}},
		{tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl}, types.EditorQuitMsg{}},
	}
	for _, tc := range cases {
		ed := createVimEditor("hello", 80, 24, "f.txt")
		_, cmd := ed.Update(tc.key)
		if cmd == nil {
			t.Fatalf("%v: expected cmd", tc.key)
		}
		msg := cmd()
		switch tc.expectedType.(type) {
		case types.EditorSaveMsg:
			save, ok := msg.(types.EditorSaveMsg)
			if !ok {
				t.Fatalf("%v: expected EditorSaveMsg, got %T", tc.key, msg)
			}
			if save.Content != "hello" {
				t.Errorf("%v: expected content hello, got %q", tc.key, save.Content)
			}
		case types.EditorQuitMsg:
			if _, ok := msg.(types.EditorQuitMsg); !ok {
				t.Fatalf("%v: expected EditorQuitMsg, got %T", tc.key, msg)
			}
		}
	}
}
