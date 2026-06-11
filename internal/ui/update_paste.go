package ui

import (
	"time"

	"charm.land/bubbles/v2/textinput"
	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

// handlePaste routes bracketed-paste text to the focused input on the current screen.
func (m Model) handlePaste(msg tea.PasteMsg) (tea.Model, tea.Cmd) {
	switch m.Screen {
	case types.ScreenAddConnection, types.ScreenEditConnection:
		return m, pasteIntoSlice(m.ConnInputs, msg)
	case types.ScreenKeys:
		if !m.Inputs.PatternInput.Focused() {
			return m, nil
		}
		inputCmd := pasteInto(&m.Inputs.PatternInput, msg)
		m.SearchSeq++
		seq := m.SearchSeq
		debounceCmd := tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
			return types.SearchDebounceMsg{Seq: seq}
		})
		return m, tea.Batch(inputCmd, debounceCmd)
	case types.ScreenAddKey:
		return m, pasteIntoSlice(m.AddKeyInputs, msg)
	case types.ScreenAddToCollection:
		return m, pasteIntoSlice(m.AddCollectionInput, msg)
	case types.ScreenPubSub, types.ScreenPublishMessage:
		return m, pasteIntoSlice(m.PubSubInput, msg)
	case types.ScreenEditValue:
		if m.VimEditor == nil {
			return m, nil
		}
		updated, cmd := m.VimEditor.Update(msg)
		m.VimEditor = updated
		return m, cmd
	case types.ScreenTTLEditor:
		return m, pasteInto(&m.Inputs.TTLInput, msg)
	case types.ScreenRenameKey:
		return m, pasteInto(&m.Inputs.RenameInput, msg)
	case types.ScreenCopyKey:
		return m, pasteInto(&m.Inputs.CopyInput, msg)
	case types.ScreenSwitchDB:
		return m, pasteInto(&m.Inputs.DBSwitchInput, msg)
	case types.ScreenSearchValues:
		return m, pasteInto(&m.Inputs.SearchValueInput, msg)
	case types.ScreenExport:
		return m, pasteInto(&m.Inputs.ExportInput, msg)
	case types.ScreenImport:
		return m, pasteInto(&m.Inputs.ImportInput, msg)
	case types.ScreenLuaScript:
		return m, pasteInto(&m.Inputs.LuaScriptInput, msg)
	case types.ScreenBulkDelete:
		return m, pasteInto(&m.Inputs.BulkDeleteInput, msg)
	case types.ScreenBatchTTL:
		return m, tea.Batch(
			pasteInto(&m.Inputs.BatchTTLInput, msg),
			pasteInto(&m.Inputs.BatchTTLPattern, msg),
		)
	case types.ScreenRegexSearch:
		return m, pasteInto(&m.Inputs.RegexSearchInput, msg)
	case types.ScreenFuzzySearch:
		return m, pasteInto(&m.Inputs.FuzzySearchInput, msg)
	case types.ScreenCompareKeys:
		return m, tea.Batch(
			pasteInto(&m.Inputs.CompareKey1Input, msg),
			pasteInto(&m.Inputs.CompareKey2Input, msg),
		)
	case types.ScreenJSONPath:
		return m, pasteInto(&m.Inputs.JSONPathInput, msg)
	case types.ScreenRedisConfig:
		return m, pasteInto(&m.Inputs.ConfigEditInput, msg)
	}
	return m, nil
}

func pasteInto(input *textinput.Model, msg tea.PasteMsg) tea.Cmd {
	updated, cmd := input.Update(msg)
	*input = updated
	return cmd
}

func pasteIntoSlice(inputs []textinput.Model, msg tea.PasteMsg) tea.Cmd {
	cmds := make([]tea.Cmd, len(inputs))
	for i := range inputs {
		inputs[i], cmds[i] = inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}
