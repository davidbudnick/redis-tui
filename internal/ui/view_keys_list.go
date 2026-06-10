package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) viewKeys() string {
	// Calculate panel widths - left panel for keys, right panel for preview
	totalWidth := m.Width
	if totalWidth < 100 {
		// If terminal is too narrow, just show keys list without preview
		return m.viewKeysListOnly()
	}

	// Left panel gets 60%, right panel gets 40%
	leftWidth := (totalWidth * 60) / 100
	rightWidth := totalWidth - leftWidth - 1

	// Build left panel (keys list)
	leftContent := m.buildKeysListPanel(leftWidth - 2)

	// Build right panel (preview)
	rightContent := m.buildPreviewPanel(rightWidth - 2)

	// Create border styles
	leftPanelStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Height(m.Height-4).
		Padding(0, 1)

	rightPanelStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(m.Height-4).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("240"))

	leftPanel := leftPanelStyle.Render(leftContent)
	rightPanel := rightPanelStyle.Render(rightContent)

	// Join panels horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Add help text at the bottom
	helpText := helpStyle.Render("j/k:nav  enter:view  a:add  d:del  /:filter  O:logs  i:info  q:back")

	return content + "\n" + helpText
}

func (m Model) viewKeysListOnly() string {
	var b strings.Builder

	connInfo := ""
	if m.CurrentConn != nil {
		if m.CurrentConn.UseCluster {
			nodeCount := len(m.ClusterNodes)
			connInfo = fmt.Sprintf(" - %s (%s:%d) cluster (%d nodes)",
				m.CurrentConn.Name, m.CurrentConn.Host, m.CurrentConn.Port, nodeCount)
		} else {
			connInfo = fmt.Sprintf(" - %s (%s:%d/db%d)", m.CurrentConn.Name, m.CurrentConn.Host, m.CurrentConn.Port, m.CurrentConn.DB)
		}
	}

	titleText := "Keys" + connInfo
	if m.TotalKeys > 0 {
		titleText += fmt.Sprintf("  [Total: %d]", m.TotalKeys)
	}
	b.WriteString(titleStyle.Render(titleText))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Filter: "))
	if m.Inputs.PatternInput.Focused() {
		b.WriteString(m.Inputs.PatternInput.View())
	} else {
		pattern := m.KeyPattern
		if pattern == "" {
			pattern = "*"
		}
		b.WriteString(normalStyle.Render(pattern))
	}
	b.WriteString("\n\n")

	if len(m.Keys) == 0 {
		b.WriteString(dimStyle.Render("No keys found."))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Press 'a' to add one."))
	} else {
		header := fmt.Sprintf("  %-40s %-12s %-15s", "Key", "Type", "TTL")
		b.WriteString(headerStyle.Render(header))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(strings.Repeat("─", 70)))
		b.WriteString("\n")

		maxVisible := m.Height - 15
		if maxVisible < 5 {
			maxVisible = 5
		}

		// Ensure selected index is within bounds
		selectedIdx := m.SelectedKeyIdx
		if selectedIdx >= len(m.Keys) {
			selectedIdx = len(m.Keys) - 1
		}
		if selectedIdx < 0 {
			selectedIdx = 0
		}

		startIdx := 0
		if selectedIdx >= maxVisible {
			startIdx = selectedIdx - maxVisible + 1
		}
		endIdx := startIdx + maxVisible
		if endIdx > len(m.Keys) {
			endIdx = len(m.Keys)
			// Adjust startIdx to show more items when at end of list
			if endIdx-startIdx < maxVisible {
				startIdx = endIdx - maxVisible
				if startIdx < 0 {
					startIdx = 0
				}
			}
		}

		for i := startIdx; i < endIdx; i++ {
			key := m.Keys[i]
			keyName := key.Key
			if len(keyName) > 40 {
				keyName = keyName[:37] + "..."
			}

			ttlStr := "∞"
			var ttlStyleLocal lipgloss.Style
			if key.TTL > 0 {
				seconds := int(key.TTL.Seconds() + 0.5)
				ttlStr = fmt.Sprintf("%ds", seconds)
				if seconds <= 10 {
					ttlStyleLocal = ttlCriticalStyle
				} else if seconds <= 60 {
					ttlStyleLocal = ttlWarningStyle
				} else {
					ttlStyleLocal = normalStyle
				}
			} else if key.TTL == -2 {
				ttlStr = "expired"
				ttlStyleLocal = errorStyle
			} else {
				ttlStyleLocal = dimStyle
			}

			typePart := getTypeStyle(key.Type).Render(fmt.Sprintf("%-12s", key.Type))
			ttlPart := ttlStyleLocal.Render(fmt.Sprintf("%-15s", ttlStr))

			line := fmt.Sprintf("%-40s %s %s", keyName, typePart, ttlPart)
			if i == selectedIdx {
				b.WriteString(selectedStyle.Render("▶ " + fmt.Sprintf("%-40s %-12s %-15s", keyName, key.Type, ttlStr)))
			} else {
				b.WriteString(normalStyle.Render("  ") + line)
			}
			b.WriteString("\n")
		}

		if len(m.Keys) > maxVisible {
			b.WriteString(dimStyle.Render(fmt.Sprintf("\nShowing %d-%d of %d", startIdx+1, endIdx, len(m.Keys))))
		}
		if m.KeyCursor > 0 {
			b.WriteString(dimStyle.Render(" [l:load more]"))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("j/k:nav  enter:view  a:add  d:del  /:filter  O:logs  i:info  q:back"))

	return b.String()
}

func (m Model) buildKeysListPanel(width int) string {
	var b strings.Builder

	// Title with connection info
	connInfo := ""
	if m.CurrentConn != nil {
		if m.CurrentConn.UseCluster {
			nodeCount := len(m.ClusterNodes)
			connInfo = fmt.Sprintf(" - %s cluster (%d nodes)", m.CurrentConn.Name, nodeCount)
		} else {
			connInfo = fmt.Sprintf(" - %s", m.CurrentConn.Name)
		}
	}
	titleText := "Keys" + connInfo
	if m.TotalKeys > 0 {
		titleText += fmt.Sprintf(" [%d]", m.TotalKeys)
	}
	b.WriteString(titleStyle.Render(titleText))
	b.WriteString("\n\n")

	// Pattern filter
	b.WriteString(keyStyle.Render("Filter: "))
	if m.Inputs.PatternInput.Focused() {
		b.WriteString(m.Inputs.PatternInput.View())
	} else {
		pattern := m.KeyPattern
		if pattern == "" {
			pattern = "*"
		}
		b.WriteString(normalStyle.Render(pattern))
	}
	b.WriteString("\n\n")

	if len(m.Keys) == 0 {
		b.WriteString(dimStyle.Render("No keys found."))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Press 'a' to add one."))
		return b.String()
	}

	// Calculate column widths
	typeWidth := 12
	ttlWidth := 6
	keyWidth := width - typeWidth - ttlWidth - 6 // 6 for spacing and cursor
	if keyWidth < 20 {
		keyWidth = 20
	}

	// Header
	header := fmt.Sprintf("  %-*s  %-*s  %s", keyWidth, "Key", typeWidth, "Type", "TTL")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	// Calculate visible range
	maxVisible := m.Height - 12
	if maxVisible < 5 {
		maxVisible = 5
	}

	// Ensure selected index is within bounds
	selectedIdx := m.SelectedKeyIdx
	if selectedIdx >= len(m.Keys) {
		selectedIdx = len(m.Keys) - 1
	}
	if selectedIdx < 0 {
		selectedIdx = 0
	}

	startIdx := 0
	if selectedIdx >= maxVisible {
		startIdx = selectedIdx - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(m.Keys) {
		endIdx = len(m.Keys)
		// Adjust startIdx to show more items when at end of list
		if endIdx-startIdx < maxVisible {
			startIdx = endIdx - maxVisible
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	// Render keys
	for i := startIdx; i < endIdx; i++ {
		key := m.Keys[i]

		// Truncate key name if needed
		keyName := key.Key
		if len(keyName) > keyWidth {
			keyName = keyName[:keyWidth-3] + "..."
		}

		// Format TTL
		ttlStr := "∞"
		if key.TTL > 0 {
			seconds := int(key.TTL.Seconds() + 0.5)
			if seconds < 60 {
				ttlStr = fmt.Sprintf("%ds", seconds)
			} else if seconds < 3600 {
				ttlStr = fmt.Sprintf("%dm", seconds/60)
			} else {
				ttlStr = fmt.Sprintf("%dh", seconds/3600)
			}
		} else if key.TTL == -2 {
			ttlStr = "exp"
		}

		if i == selectedIdx {
			// Selected row - highlight entire row
			typeStyled := getTypeStyleBold(key.Type).Render(fmt.Sprintf("%-*s", typeWidth, key.Type))

			cursor := selectedStyle.Render("▶ ")
			keyPart := selectedStyle.Render(fmt.Sprintf("%-*s", keyWidth, keyName))

			b.WriteString(cursor)
			b.WriteString(keyPart)
			b.WriteString("  ")
			b.WriteString(typeStyled)
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(ttlStr))
		} else {
			// Normal row
			typeStyled := getTypeStyle(key.Type).Render(fmt.Sprintf("%-*s", typeWidth, key.Type))

			b.WriteString("  ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-*s", keyWidth, keyName)))
			b.WriteString("  ")
			b.WriteString(typeStyled)
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(ttlStr))
		}
		b.WriteString("\n")
	}

	// Show pagination info
	if len(m.Keys) > maxVisible {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("%d-%d of %d", startIdx+1, endIdx, len(m.Keys))))
		if m.KeyCursor > 0 {
			b.WriteString(dimStyle.Render(" • l:more"))
		}
	}

	return b.String()
}
