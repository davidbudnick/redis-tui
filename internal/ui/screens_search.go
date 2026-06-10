package ui

import (
	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

func (m Model) handleSearchValuesScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.Inputs.SearchValueInput.Value() != "" {
			m.Loading = true
			pattern := m.KeyPattern
			if pattern == "" {
				pattern = "*"
			}
			m.Screen = types.ScreenKeys
			return m, m.Cmds.SearchByValue(pattern, m.Inputs.SearchValueInput.Value(), 100)
		}
	case "esc":
		m.Screen = types.ScreenKeys
		m.Inputs.SearchValueInput.Blur()
	default:
		var inputCmd tea.Cmd
		m.Inputs.SearchValueInput, inputCmd = m.Inputs.SearchValueInput.Update(msg)
		return m, inputCmd
	}
	return m, nil
}

func (m Model) handleRegexSearchScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.Inputs.RegexSearchInput.Value() != "" {
			m.Loading = true
			return m, m.Cmds.RegexSearch(m.Inputs.RegexSearchInput.Value(), 100)
		}
	case "esc":
		m.Screen = types.ScreenKeys
		m.Inputs.RegexSearchInput.Blur()
	default:
		var inputCmd tea.Cmd
		m.Inputs.RegexSearchInput, inputCmd = m.Inputs.RegexSearchInput.Update(msg)
		return m, inputCmd
	}
	return m, nil
}

func (m Model) handleFuzzySearchScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.Inputs.FuzzySearchInput.Value() != "" {
			m.Loading = true
			return m, m.Cmds.FuzzySearch(m.Inputs.FuzzySearchInput.Value(), 100)
		}
	case "esc":
		m.Screen = types.ScreenKeys
		m.Inputs.FuzzySearchInput.Blur()
	default:
		var inputCmd tea.Cmd
		m.Inputs.FuzzySearchInput, inputCmd = m.Inputs.FuzzySearchInput.Update(msg)
		return m, inputCmd
	}
	return m, nil
}

func (m Model) handleCompareKeysScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		if m.CompareFocusIdx == 0 {
			m.Inputs.CompareKey1Input.Blur()
			m.Inputs.CompareKey2Input.Focus()
			m.CompareFocusIdx = 1
		} else {
			m.Inputs.CompareKey2Input.Blur()
			m.Inputs.CompareKey1Input.Focus()
			m.CompareFocusIdx = 0
		}
	case "enter":
		if m.Inputs.CompareKey1Input.Value() != "" && m.Inputs.CompareKey2Input.Value() != "" {
			m.Loading = true
			return m, m.Cmds.CompareKeys(m.Inputs.CompareKey1Input.Value(), m.Inputs.CompareKey2Input.Value())
		}
	case "esc":
		m.Screen = types.ScreenKeys
		m.Inputs.CompareKey1Input.Blur()
		m.Inputs.CompareKey2Input.Blur()
		m.CompareResult = nil
	default:
		if m.CompareFocusIdx == 0 {
			var inputCmd tea.Cmd
			m.Inputs.CompareKey1Input, inputCmd = m.Inputs.CompareKey1Input.Update(msg)
			return m, inputCmd
		}
		var inputCmd tea.Cmd
		m.Inputs.CompareKey2Input, inputCmd = m.Inputs.CompareKey2Input.Update(msg)
		return m, inputCmd
	}
	return m, nil
}

func (m Model) handleJSONPathScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.Inputs.JSONPathInput.Value() != "" && m.CurrentKey != nil {
			m.Loading = true
			return m, m.Cmds.JSONPathQuery(m.CurrentKey.Key, m.Inputs.JSONPathInput.Value())
		}
	case "esc":
		m.Screen = types.ScreenKeyDetail
		m.Inputs.JSONPathInput.Blur()
	default:
		var inputCmd tea.Cmd
		m.Inputs.JSONPathInput, inputCmd = m.Inputs.JSONPathInput.Update(msg)
		return m, inputCmd
	}
	return m, nil
}
