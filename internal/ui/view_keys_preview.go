package ui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func (m Model) buildPreviewPanel(width int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Preview"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n\n")

	// Check if we have keys
	if len(m.Keys) == 0 || m.SelectedKeyIdx >= len(m.Keys) {
		b.WriteString(dimStyle.Render("No key selected"))
		return b.String()
	}

	selectedKey := m.Keys[m.SelectedKeyIdx]

	// Key name (with word wrap for long keys)
	b.WriteString(keyStyle.Render("Key: "))
	keyName := selectedKey.Key
	if width > 12 && len(keyName) > width-6 {
		keyName = keyName[:width-9] + "..."
	}
	b.WriteString(normalStyle.Render(keyName))
	b.WriteString("\n\n")

	// Type with color
	b.WriteString(keyStyle.Render("Type: "))
	b.WriteString(getTypeStyleBold(selectedKey.Type).Render(string(selectedKey.Type)))
	b.WriteString("\n\n")

	// TTL with visual indicator
	b.WriteString(keyStyle.Render("TTL: "))
	if selectedKey.TTL > 0 {
		seconds := int(selectedKey.TTL.Seconds() + 0.5)
		var ttlStr string
		var ttlStyle lipgloss.Style

		if seconds < 60 {
			ttlStr = fmt.Sprintf("%d seconds", seconds)
		} else if seconds < 3600 {
			ttlStr = fmt.Sprintf("%d minutes", seconds/60)
		} else if seconds < 86400 {
			ttlStr = fmt.Sprintf("%d hours", seconds/3600)
		} else {
			ttlStr = fmt.Sprintf("%d days", seconds/86400)
		}

		if seconds <= 10 {
			ttlStyle = errorStyle // Red
			ttlStr = "⚠ " + ttlStr
		} else if seconds <= 60 {
			ttlStyle = ttlWarningStyle // Yellow
		} else {
			ttlStyle = ttlGreenStyle // Green
		}
		b.WriteString(ttlStyle.Render(ttlStr))
	} else {
		b.WriteString(dimStyle.Render("No expiry"))
	}
	b.WriteString("\n\n")

	// Separator before value
	b.WriteString(dimStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n\n")

	// Value section
	b.WriteString(keyStyle.Render("Value"))
	b.WriteString("\n\n")

	// Check if preview is loaded
	if m.PreviewKey != selectedKey.Key {
		b.WriteString(dimStyle.Render("Loading..."))
		return b.String()
	}

	// Render value based on type
	maxLines := m.Height - 20
	if maxLines < 5 {
		maxLines = 5
	}

	valueContent := m.formatPreviewValue(width, maxLines)
	b.WriteString(valueContent)

	return b.String()
}

func (m Model) formatPreviewValue(maxWidth, maxLines int) string {
	if maxWidth < 15 {
		return dimStyle.Render("(too narrow)")
	}
	var lines []string

	switch m.PreviewValue.Type {
	case types.KeyTypeString:
		value := m.PreviewValue.StringValue
		formatted := formatPossibleJSON(value)

		// Split into lines and limit
		valueLines := strings.Split(formatted, "\n")
		for i, line := range valueLines {
			if i >= maxLines {
				lines = append(lines, dimStyle.Render(fmt.Sprintf("... (%d more lines)", len(valueLines)-i)))
				break
			}
			if len(line) > maxWidth {
				line = line[:maxWidth-3] + "..."
			}
			lines = append(lines, normalStyle.Render(line))
		}

	case types.KeyTypeList:
		if len(m.PreviewValue.ListValue) == 0 {
			return dimStyle.Render("(empty list)")
		}
		lines = append(lines, dimStyle.Render(fmt.Sprintf("Length: %d", len(m.PreviewValue.ListValue))))
		lines = append(lines, "")

		for i, v := range m.PreviewValue.ListValue {
			if i >= maxLines-2 {
				lines = append(lines, dimStyle.Render(fmt.Sprintf("... and %d more items", len(m.PreviewValue.ListValue)-i)))
				break
			}
			val := v
			if len(val) > maxWidth-8 {
				val = val[:maxWidth-11] + "..."
			}
			idx := dimStyle.Render(fmt.Sprintf("[%d]", i))
			lines = append(lines, fmt.Sprintf("%s %s", idx, normalStyle.Render(val)))
		}

	case types.KeyTypeSet:
		if len(m.PreviewValue.SetValue) == 0 {
			return dimStyle.Render("(empty set)")
		}
		lines = append(lines, dimStyle.Render(fmt.Sprintf("Members: %d", len(m.PreviewValue.SetValue))))
		lines = append(lines, "")

		for i, v := range m.PreviewValue.SetValue {
			if i >= maxLines-2 {
				lines = append(lines, dimStyle.Render(fmt.Sprintf("... and %d more", len(m.PreviewValue.SetValue)-i)))
				break
			}
			val := v
			if len(val) > maxWidth-4 {
				val = val[:maxWidth-7] + "..."
			}
			lines = append(lines, normalStyle.Render("• "+val))
		}

	case types.KeyTypeZSet:
		if len(m.PreviewValue.ZSetValue) == 0 {
			return dimStyle.Render("(empty sorted set)")
		}
		lines = append(lines, dimStyle.Render(fmt.Sprintf("Members: %d", len(m.PreviewValue.ZSetValue))))
		lines = append(lines, "")

		for i, v := range m.PreviewValue.ZSetValue {
			if i >= maxLines-2 {
				lines = append(lines, dimStyle.Render(fmt.Sprintf("... and %d more", len(m.PreviewValue.ZSetValue)-i)))
				break
			}
			member := v.Member
			if len(member) > maxWidth-12 {
				member = member[:maxWidth-15] + "..."
			}
			score := zsetScoreStyle.Render(fmt.Sprintf("%.1f", v.Score))
			lines = append(lines, fmt.Sprintf("%s → %s", score, normalStyle.Render(member)))
		}

	case types.KeyTypeHash:
		if len(m.PreviewValue.HashValue) == 0 {
			return dimStyle.Render("(empty hash)")
		}

		// Sort keys for consistent display
		hashKeys := make([]string, 0, len(m.PreviewValue.HashValue))
		for k := range m.PreviewValue.HashValue {
			hashKeys = append(hashKeys, k)
		}
		sort.Strings(hashKeys)

		lines = append(lines, dimStyle.Render(fmt.Sprintf("Fields: %d", len(hashKeys))))
		lines = append(lines, "")

		for i, k := range hashKeys {
			if i >= maxLines-2 {
				lines = append(lines, dimStyle.Render(fmt.Sprintf("... and %d more", len(hashKeys)-i)))
				break
			}
			v := m.PreviewValue.HashValue[k]

			// Truncate key and value
			displayKey := k
			if len(displayKey) > 15 {
				displayKey = displayKey[:12] + "..."
			}

			maxValLen := maxWidth - len(displayKey) - 5
			if maxValLen < 10 {
				maxValLen = 10
			}
			displayVal := v
			if len(displayVal) > maxValLen {
				displayVal = displayVal[:maxValLen-3] + "..."
			}

			fieldName := hashFieldStyle.Render(displayKey)
			lines = append(lines, fmt.Sprintf("%s: %s", fieldName, normalStyle.Render(displayVal)))
		}

	case types.KeyTypeStream:
		if len(m.PreviewValue.StreamValue) == 0 {
			return dimStyle.Render("(empty stream)")
		}
		lines = append(lines, dimStyle.Render(fmt.Sprintf("Entries: %d", len(m.PreviewValue.StreamValue))))
		lines = append(lines, "")

		for i, entry := range m.PreviewValue.StreamValue {
			if i >= maxLines-2 {
				lines = append(lines, dimStyle.Render(fmt.Sprintf("... and %d more", len(m.PreviewValue.StreamValue)-i)))
				break
			}

			// Format entry ID
			idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

			// Format fields
			var fields []string
			for k, v := range entry.Fields {
				fields = append(fields, fmt.Sprintf("%s=%v", k, v))
			}
			fieldStr := strings.Join(fields, ", ")
			if len(fieldStr) > maxWidth-20 {
				fieldStr = fieldStr[:maxWidth-23] + "..."
			}

			lines = append(lines, fmt.Sprintf("%s %s", idStyle.Render(entry.ID), dimStyle.Render(fieldStr)))
		}

	case types.KeyTypeJSON:
		value := m.PreviewValue.JSONValue
		formatted := formatPossibleJSON(value)

		valueLines := strings.Split(formatted, "\n")
		for i, line := range valueLines {
			if i >= maxLines {
				lines = append(lines, dimStyle.Render(fmt.Sprintf("... (%d more lines)", len(valueLines)-i)))
				break
			}
			if len(line) > maxWidth {
				line = line[:maxWidth-3] + "..."
			}
			lines = append(lines, normalStyle.Render(line))
		}

	case types.KeyTypeHyperLogLog:
		lines = append(lines, normalStyle.Render(fmt.Sprintf("Estimated cardinality: %d", m.PreviewValue.HLLCount)))

	case types.KeyTypeBitmap:
		lines = append(lines, dimStyle.Render(fmt.Sprintf("Bit count: %d", m.PreviewValue.BitCount)))
		lines = append(lines, "")
		if len(m.PreviewValue.BitPositions) > 0 {
			lines = append(lines, keyStyle.Render("Set positions:"))
			for i, pos := range m.PreviewValue.BitPositions {
				if i >= maxLines-4 {
					lines = append(lines, dimStyle.Render(fmt.Sprintf("... and %d more", len(m.PreviewValue.BitPositions)-i)))
					break
				}
				lines = append(lines, normalStyle.Render(fmt.Sprintf("  bit %d = 1", pos)))
			}
		} else {
			lines = append(lines, dimStyle.Render("(all bits are 0)"))
		}

	case types.KeyTypeGeo:
		if len(m.PreviewValue.GeoValue) == 0 {
			return dimStyle.Render("(empty geo set)")
		}
		lines = append(lines, dimStyle.Render(fmt.Sprintf("Members: %d", len(m.PreviewValue.GeoValue))))
		lines = append(lines, "")

		for i, g := range m.PreviewValue.GeoValue {
			if i >= maxLines-2 {
				lines = append(lines, dimStyle.Render(fmt.Sprintf("... and %d more", len(m.PreviewValue.GeoValue)-i)))
				break
			}
			name := g.Name
			if len(name) > maxWidth-25 {
				name = name[:maxWidth-28] + "..."
			}
			lines = append(lines, fmt.Sprintf("%s  %s",
				normalStyle.Render(name),
				dimStyle.Render(fmt.Sprintf("(%.4f, %.4f)", g.Longitude, g.Latitude))))
		}

	default:
		return dimStyle.Render("(unknown type)")
	}

	return strings.Join(lines, "\n")
}
