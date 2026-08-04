package ui

import (
	"charm.land/bubbles/v2/textinput"
	"github.com/davidbudnick/redis-tui/internal/types"
)

// textEntryActive reports whether the current screen is capturing free-form
// text, in which case single-letter global shortcuts (q, ?) must not fire —
// otherwise typing a filter like "sidekiq" quits the app.
func (m Model) textEntryActive() bool {
	if m.Screen == types.ScreenEditValue && m.VimEditor != nil {
		return true
	}
	_, focused := m.focusedInputValue()
	return focused
}

// focusedInputValue returns the text of the focused input on the current screen.
func (m Model) focusedInputValue() (string, bool) {
	switch m.Screen {
	case types.ScreenAddConnection, types.ScreenEditConnection:
		return focusedValue(m.ConnInputs...)
	case types.ScreenKeys:
		if m.Inputs.PatternInput.Focused() {
			return m.Inputs.PatternInput.Value(), true
		}
	case types.ScreenAddKey:
		return focusedValue(m.AddKeyInputs...)
	case types.ScreenAddToCollection:
		return focusedValue(m.AddCollectionInput...)
	case types.ScreenPubSub, types.ScreenPublishMessage:
		return focusedValue(m.PubSubInput...)
	case types.ScreenTTLEditor:
		return focusedValue(m.Inputs.TTLInput)
	case types.ScreenRenameKey:
		return focusedValue(m.Inputs.RenameInput)
	case types.ScreenCopyKey:
		return focusedValue(m.Inputs.CopyInput)
	case types.ScreenSwitchDB:
		return focusedValue(m.Inputs.DBSwitchInput)
	case types.ScreenSearchValues:
		return focusedValue(m.Inputs.SearchValueInput)
	case types.ScreenExport:
		return focusedValue(m.Inputs.ExportInput)
	case types.ScreenImport:
		return focusedValue(m.Inputs.ImportInput)
	case types.ScreenLuaScript:
		return focusedValue(m.Inputs.LuaScriptInput)
	case types.ScreenBulkDelete:
		return focusedValue(m.Inputs.BulkDeleteInput)
	case types.ScreenBatchTTL:
		return focusedValue(m.Inputs.BatchTTLInput, m.Inputs.BatchTTLPattern)
	case types.ScreenRegexSearch:
		return focusedValue(m.Inputs.RegexSearchInput)
	case types.ScreenFuzzySearch:
		return focusedValue(m.Inputs.FuzzySearchInput)
	case types.ScreenCompareKeys:
		return focusedValue(m.Inputs.CompareKey1Input, m.Inputs.CompareKey2Input)
	case types.ScreenJSONPath:
		return focusedValue(m.Inputs.JSONPathInput)
	case types.ScreenRedisConfig:
		return focusedValue(m.Inputs.ConfigEditInput)
	}
	return "", false
}

func focusedValue(inputs ...textinput.Model) (string, bool) {
	for i := range inputs {
		if inputs[i].Focused() {
			return inputs[i].Value(), true
		}
	}
	return "", false
}
