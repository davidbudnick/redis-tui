// Package editor provides a plain multiline text editor backed by bubbles textarea.
package editor

import (
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"

	"github.com/davidbudnick/redis-tui/internal/types"
)

type Model struct {
	area     textarea.Model
	fileName string
}

func New(content string, width, height int, fileName string) *Model {
	area := textarea.New()
	area.ShowLineNumbers = true
	area.CharLimit = 0
	area.SetValue(content)
	area.SetWidth(width)
	area.SetHeight(height)
	area.Focus()
	return &Model{area: area, fileName: fileName}
}

func (m *Model) FileName() string {
	return m.fileName
}

func (m *Model) Value() string {
	return m.area.Value()
}

func (m *Model) SetSize(width, height int) {
	m.area.SetWidth(width)
	m.area.SetHeight(height)
}

func (m *Model) View() string {
	return m.area.View()
}

func (m *Model) Save() tea.Cmd {
	content := m.area.Value()
	return func() tea.Msg {
		return types.EditorSaveMsg{Content: content}
	}
}

func (m *Model) Cancel() tea.Cmd {
	return func() tea.Msg {
		return types.EditorQuitMsg{}
	}
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch key := msg.(type) {
	case tea.KeyPressMsg:
		switch key.String() {
		case "ctrl+s":
			return m, m.Save()
		case "esc", "ctrl+q":
			return m, m.Cancel()
		}
	}
	area, cmd := m.area.Update(msg)
	m.area = area
	return m, cmd
}
