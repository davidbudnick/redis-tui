package ui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func (m Model) viewTTLEditor() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Set TTL"))
	b.WriteString("\n\n")

	if m.CurrentKey != nil {
		b.WriteString(keyStyle.Render("Key: "))
		b.WriteString(normalStyle.Render(m.CurrentKey.Key))
		b.WriteString("\n\n")
	}

	b.WriteString(keyStyle.Render("TTL (seconds):"))
	b.WriteString("\n")
	b.WriteString(m.Inputs.TTLInput.View())
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Enter 0 to remove expiry"))
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("enter:save  esc:cancel"))

	modalWidth := 50
	if m.Width-10 < 50 {
		modalWidth = m.Width - 10
	}
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(modalWidth)

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, modalStyle.Render(b.String()))
}

func (m Model) viewEditValue() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Edit Value"))
	if m.CurrentKey != nil {
		b.WriteString("  ")
		b.WriteString(dimStyle.Render(m.CurrentKey.Key))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("ctrl+s:save  ctrl+q:quit  :w/:q in command mode"))
	b.WriteString("\n\n")

	if m.VimEditor != nil {
		b.WriteString(m.VimEditor.View())
	}

	return b.String()
}

func (m Model) viewAddToCollection() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Add to Collection"))
	b.WriteString("\n\n")

	if m.CurrentKey != nil {
		b.WriteString(keyStyle.Render("Key: "))
		b.WriteString(normalStyle.Render(m.CurrentKey.Key))
		b.WriteString(" (")
		b.WriteString(getTypeStyle(m.CurrentKey.Type).Render(string(m.CurrentKey.Type)))
		b.WriteString(")")
		b.WriteString("\n\n")

		var label1, label2 string
		switch m.CurrentKey.Type {
		case types.KeyTypeList:
			label1, label2 = "Element:", ""
		case types.KeyTypeSet:
			label1, label2 = "Member:", ""
		case types.KeyTypeZSet:
			label1, label2 = "Member:", "Score:"
		case types.KeyTypeHash:
			label1, label2 = "Field:", "Value:"
		case types.KeyTypeStream:
			label1, label2 = "Field:", "Value:"
		case types.KeyTypeHyperLogLog:
			label1, label2 = "Element:", ""
		case types.KeyTypeBitmap:
			label1, label2 = "Offset:", ""
		case types.KeyTypeGeo:
			label1, label2 = "Member:", "Lon,Lat:"
		}

		b.WriteString(keyStyle.Render(label1))
		b.WriteString("\n")
		b.WriteString(m.AddCollectionInput[0].View())
		b.WriteString("\n\n")

		if label2 != "" {
			b.WriteString(keyStyle.Render(label2))
			b.WriteString("\n")
			b.WriteString(m.AddCollectionInput[1].View())
			b.WriteString("\n\n")
		}
	}

	b.WriteString(helpStyle.Render("tab:next  enter:add  esc:cancel"))

	return m.renderModal(b.String())
}

func (m Model) viewRemoveFromCollection() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Remove from Collection"))
	b.WriteString("\n\n")

	if m.CurrentKey != nil {
		b.WriteString(keyStyle.Render("Key: "))
		b.WriteString(normalStyle.Render(m.CurrentKey.Key))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Select item to remove:"))
		b.WriteString("\n\n")

		switch m.CurrentValue.Type {
		case types.KeyTypeList:
			for i, v := range m.CurrentValue.ListValue {
				prefix := "  "
				if i == m.SelectedItemIdx {
					prefix = "> "
					b.WriteString(selectedStyle.Render(fmt.Sprintf("%s%d: %s", prefix, i, truncate(v, 50))))
				} else {
					b.WriteString(normalStyle.Render(fmt.Sprintf("%s%d: %s", prefix, i, truncate(v, 50))))
				}
				b.WriteString("\n")
			}
		case types.KeyTypeSet:
			for i, v := range m.CurrentValue.SetValue {
				prefix := "  "
				if i == m.SelectedItemIdx {
					b.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", truncate(v, 50))))
				} else {
					b.WriteString(normalStyle.Render(fmt.Sprintf("%s%s", prefix, truncate(v, 50))))
				}
				b.WriteString("\n")
			}
		case types.KeyTypeZSet:
			for i, v := range m.CurrentValue.ZSetValue {
				if i == m.SelectedItemIdx {
					b.WriteString(selectedStyle.Render(fmt.Sprintf("> %.2f: %s", v.Score, truncate(v.Member, 45))))
				} else {
					b.WriteString(normalStyle.Render(fmt.Sprintf("  %.2f: %s", v.Score, truncate(v.Member, 45))))
				}
				b.WriteString("\n")
			}
		case types.KeyTypeHash:
			keys := make([]string, 0, len(m.CurrentValue.HashValue))
			for k := range m.CurrentValue.HashValue {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for i, k := range keys {
				v := m.CurrentValue.HashValue[k]
				if i == m.SelectedItemIdx {
					b.WriteString(selectedStyle.Render(fmt.Sprintf("> %s: %s", k, truncate(v, 40))))
				} else {
					b.WriteString(normalStyle.Render(fmt.Sprintf("  %s: %s", k, truncate(v, 40))))
				}
				b.WriteString("\n")
			}
		case types.KeyTypeStream:
			for i, e := range m.CurrentValue.StreamValue {
				if i == m.SelectedItemIdx {
					b.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", e.ID)))
				} else {
					b.WriteString(normalStyle.Render(fmt.Sprintf("  %s", e.ID)))
				}
				b.WriteString("\n")
			}
		case types.KeyTypeGeo:
			for i, g := range m.CurrentValue.GeoValue {
				display := fmt.Sprintf("%s  (%.4f, %.4f)", g.Name, g.Longitude, g.Latitude)
				if i == m.SelectedItemIdx {
					b.WriteString(selectedStyle.Render("> " + display))
				} else {
					b.WriteString(normalStyle.Render("  " + display))
				}
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:navigate  enter/d:delete  esc:cancel"))

	return m.renderModal(b.String())
}

func (m Model) viewRenameKey() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Rename Key"))
	b.WriteString("\n\n")

	if m.CurrentKey != nil {
		b.WriteString(keyStyle.Render("Current: "))
		b.WriteString(normalStyle.Render(m.CurrentKey.Key))
		b.WriteString("\n\n")
	}

	b.WriteString(keyStyle.Render("New Name:"))
	b.WriteString("\n")
	b.WriteString(m.Inputs.RenameInput.View())
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("enter:rename  esc:cancel"))

	return m.renderModal(b.String())
}

func (m Model) viewCopyKey() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Copy Key"))
	b.WriteString("\n\n")

	if m.CurrentKey != nil {
		b.WriteString(keyStyle.Render("Source: "))
		b.WriteString(normalStyle.Render(m.CurrentKey.Key))
		b.WriteString("\n\n")
	}

	b.WriteString(keyStyle.Render("Destination:"))
	b.WriteString("\n")
	b.WriteString(m.Inputs.CopyInput.View())
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("enter:copy  esc:cancel"))

	return m.renderModal(b.String())
}
