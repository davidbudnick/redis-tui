package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestHandleSearchValuesScreen(t *testing.T) {
	t.Run("enter valid", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.SearchValueInput.SetValue("query")
		m.KeyPattern = "user:*"
		_, cmd := m.handleSearchValuesScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter empty pattern", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.SearchValueInput.SetValue("query")
		_, cmd := m.handleSearchValuesScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter empty input", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleSearchValuesScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleSearchValuesScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
	t.Run("default", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleSearchValuesScreen(keyMsg('a'))
	})
}

func TestHandleRegexSearchScreen(t *testing.T) {
	t.Run("enter valid", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.RegexSearchInput.SetValue("^foo.*")
		_, cmd := m.handleRegexSearchScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleRegexSearchScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleRegexSearchScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
	t.Run("default", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleRegexSearchScreen(keyMsg('a'))
	})
}

func TestHandleFuzzySearchScreen(t *testing.T) {
	t.Run("enter valid", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.FuzzySearchInput.SetValue("foo")
		_, cmd := m.handleFuzzySearchScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleFuzzySearchScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleFuzzySearchScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
	t.Run("default", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleFuzzySearchScreen(keyMsg('a'))
	})
}

func TestHandleCompareKeysScreen(t *testing.T) {
	t.Run("tab from 0", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CompareFocusIdx = 0
		result, _ := m.handleCompareKeysScreen(tea.KeyPressMsg{Code: tea.KeyTab})
		if result.(Model).CompareFocusIdx != 1 {
			t.Errorf("expected 1, got %d", result.(Model).CompareFocusIdx)
		}
	})
	t.Run("tab from 1", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CompareFocusIdx = 1
		result, _ := m.handleCompareKeysScreen(tea.KeyPressMsg{Code: tea.KeyTab})
		if result.(Model).CompareFocusIdx != 0 {
			t.Errorf("expected 0, got %d", result.(Model).CompareFocusIdx)
		}
	})
	t.Run("enter valid", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.CompareKey1Input.SetValue("k1")
		m.Inputs.CompareKey2Input.SetValue("k2")
		_, cmd := m.handleCompareKeysScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleCompareKeysScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CompareResult = &types.KeyComparison{}
		result, _ := m.handleCompareKeysScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		if result.(Model).CompareResult != nil {
			t.Error("expected result cleared")
		}
	})
	t.Run("default to key1", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CompareFocusIdx = 0
		_, _ = m.handleCompareKeysScreen(keyMsg('a'))
	})
	t.Run("default to key2", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CompareFocusIdx = 1
		_, _ = m.handleCompareKeysScreen(keyMsg('a'))
	})
}

func TestHandleJSONPathScreen(t *testing.T) {
	t.Run("enter valid", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		m.Inputs.JSONPathInput.SetValue("$.a")
		_, cmd := m.handleJSONPathScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "foo"}
		_, cmd := m.handleJSONPathScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("enter no key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.JSONPathInput.SetValue("$.a")
		_, cmd := m.handleJSONPathScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleJSONPathScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
	t.Run("default", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleJSONPathScreen(keyMsg('a'))
	})
}
