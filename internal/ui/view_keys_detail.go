package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func (m Model) viewKeyDetail() string {
	var b strings.Builder

	if m.CurrentKey == nil {
		return "No key selected"
	}

	boxWidth := detailBoxWidth(m.Width)
	contentWidth := detailContentWidth(boxWidth)

	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, titleStyle.Render("Key Detail")))
	b.WriteString("\n\n")

	// Build metadata block and center it
	var meta strings.Builder
	meta.WriteString(keyStyle.Render("    Key: "))
	meta.WriteString(normalStyle.Render(m.CurrentKey.Key))
	meta.WriteString("\n")

	meta.WriteString(keyStyle.Render("   Type: "))
	meta.WriteString(getTypeStyleBold(m.CurrentKey.Type).Render(string(m.CurrentKey.Type)))
	meta.WriteString("\n")

	meta.WriteString(keyStyle.Render("    TTL: "))
	ttlStr := "No expiry"
	var ttlDetailStyle lipgloss.Style
	if m.CurrentKey.TTL > 0 {
		seconds := int(m.CurrentKey.TTL.Seconds() + 0.5) // round to nearest second
		ttlStr = fmt.Sprintf("%ds", seconds)
		if seconds <= 10 {
			ttlDetailStyle = ttlCriticalStyle
			ttlStr = "⚠ " + ttlStr
		} else if seconds <= 60 {
			ttlDetailStyle = ttlWarningStyle
			ttlStr = "⏱ " + ttlStr
		} else {
			ttlDetailStyle = normalStyle
		}
	} else {
		ttlDetailStyle = dimStyle
	}
	meta.WriteString(ttlDetailStyle.Render(ttlStr))

	// Show memory usage if available
	if m.MemoryUsage > 0 {
		meta.WriteString("  ")
		meta.WriteString(keyStyle.Render("Memory: "))
		meta.WriteString(normalStyle.Render(formatBytes(m.MemoryUsage)))
	}

	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, meta.String()))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, keyStyle.Render("Value:")))
	b.WriteString("\n")

	valueBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(boxWidth)

	valueStr := m.detailValueContent()
	// Wrap to the box content width so scroll line counts match what is painted.
	valueLines := wrapPlainLines(strings.Split(valueStr, "\n"), contentWidth)
	maxVisible := detailMaxVisible(m.Height)

	// Derive a display scroll that keeps the cursor visible without mutating model state
	// (View receives Model by value).
	_, displayScroll := ensureDetailCursorVisible(m.DetailCursor, m.DetailScroll, len(valueLines), maxVisible)
	visible, topHint, bottomHint, displayScroll := scrollValueLines(valueLines, displayScroll, maxVisible)

	var display strings.Builder
	if topHint != "" {
		display.WriteString(topHint)
		display.WriteByte('\n')
	}
	for i, line := range visible {
		abs := displayScroll + i
		if abs == m.DetailCursor {
			// Full-width blue selection band (plain text, high contrast).
			plain := padRight(truncateRunes(line, contentWidth), contentWidth)
			display.WriteString(selectedStyle.Render(plain))
		} else if m.CurrentValue.Type == types.KeyTypeProtobuf {
			display.WriteString(colorizeProtobufLine(line))
		} else {
			display.WriteString(line)
		}
		if i < len(visible)-1 {
			display.WriteByte('\n')
		}
	}
	if bottomHint != "" {
		display.WriteByte('\n')
		display.WriteString(bottomHint)
	}

	b.WriteString(valueBox.Render(display.String()))
	b.WriteString("\n\n")

	helpText := "j/k:move  t:TTL  d:del  r:refresh  R:rename  c:copy"
	switch m.CurrentKey.Type {
	case types.KeyTypeJSON:
		helpText += "  e:edit"
	case types.KeyTypeString:
		if stringValueEditable(m.CurrentValue.StringValue) {
			helpText += "  e:edit"
		}
	case types.KeyTypeHyperLogLog, types.KeyTypeBitmap:
		helpText += "  a:add"
	case types.KeyTypeProtobuf:
		// read-only decoded view
	default:
		helpText += "  a:add  x:remove"
	}
	helpText += "  esc:back"
	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, helpStyle.Render(helpText)))

	return b.String()
}

func (m Model) detailContentLines() int {
	lines := wrapPlainLines(strings.Split(m.detailValueContent(), "\n"), detailContentWidth(detailBoxWidth(m.Width)))
	return len(lines)
}

// detailLastLine is the last selectable line index in the detail value body.
func (m Model) detailLastLine() int {
	return max(m.detailContentLines()-1, 0)
}

// detailValueContent returns the formatted value block, using the per-load cache when set.
func (m Model) detailValueContent() string {
	if m.DetailRendered != "" {
		return m.DetailRendered
	}
	return buildDetailValueContent(m.CurrentValue)
}

// detailValueString returns the detail body used by tests and line counting.
func (m Model) detailValueString() string {
	return m.detailValueContent()
}

// detailValuePlainString builds the unstyled value body for the detail pane.
func (m Model) detailValuePlainString() string {
	return buildDetailValueContent(m.CurrentValue)
}

// buildDetailValueContent formats a Redis value for the key detail view.
func buildDetailValueContent(value types.RedisValue) string {
	var vc strings.Builder
	switch value.Type {
	case types.KeyTypeString:
		vc.WriteString(formatPossibleJSON(value.StringValue))
	case types.KeyTypeList:
		if len(value.ListValue) == 0 {
			vc.WriteString("(empty list)")
		} else {
			for i, v := range value.ListValue {
				fmt.Fprintf(&vc, "%d. %s\n", i, formatPossibleJSON(v))
			}
		}
	case types.KeyTypeSet:
		if len(value.SetValue) == 0 {
			vc.WriteString("(empty set)")
		} else {
			for _, v := range value.SetValue {
				vc.WriteString("• ")
				vc.WriteString(formatPossibleJSON(v))
				vc.WriteString("\n")
			}
		}
	case types.KeyTypeZSet:
		if len(value.ZSetValue) == 0 {
			vc.WriteString("(empty sorted set)")
		} else {
			for _, v := range value.ZSetValue {
				fmt.Fprintf(&vc, "%.2f: %s\n", v.Score, formatPossibleJSON(v.Member))
			}
		}
	case types.KeyTypeHash:
		if len(value.HashValue) == 0 {
			vc.WriteString("(empty hash)")
		} else {
			hashKeys := make([]string, 0, len(value.HashValue))
			for k := range value.HashValue {
				hashKeys = append(hashKeys, k)
			}
			sort.Strings(hashKeys)
			for _, k := range hashKeys {
				v := value.HashValue[k]
				formattedValue := formatPossibleJSON(v)
				if strings.Contains(formattedValue, "\n") {
					fmt.Fprintf(&vc, "◆ %s:\n%s\n", k, formattedValue)
				} else {
					fmt.Fprintf(&vc, "◆ %s: %s\n", k, formattedValue)
				}
			}
		}
	case types.KeyTypeStream:
		if len(value.StreamValue) == 0 {
			vc.WriteString("(empty stream)")
		} else {
			for _, entry := range value.StreamValue {
				jsonBytes, err := json.MarshalIndent(entry.Fields, "", "  ")
				if err == nil {
					fmt.Fprintf(&vc, "%s:\n%s\n", entry.ID, string(jsonBytes))
				} else {
					fields := []string{}
					for k, v := range entry.Fields {
						fields = append(fields, fmt.Sprintf("%s=%v", k, v))
					}
					fmt.Fprintf(&vc, "%s: %s\n", entry.ID, strings.Join(fields, ", "))
				}
			}
		}
	case types.KeyTypeJSON:
		vc.WriteString(formatPossibleJSON(value.JSONValue))
	case types.KeyTypeHyperLogLog:
		fmt.Fprintf(&vc, "Estimated cardinality: %d", value.HLLCount)
	case types.KeyTypeProtobuf:
		vc.WriteString(formatProtobufValue(value))
	case types.KeyTypeBitmap:
		fmt.Fprintf(&vc, "Bit count: %d\n\n", value.BitCount)
		if len(value.BitPositions) > 0 {
			vc.WriteString("Set positions:\n")
			for _, pos := range value.BitPositions {
				fmt.Fprintf(&vc, "  bit %d = 1\n", pos)
			}
			if value.BitCount > int64(len(value.BitPositions)) {
				fmt.Fprintf(&vc, "  … and %d more\n", value.BitCount-int64(len(value.BitPositions)))
			}
		} else {
			vc.WriteString("(all bits are 0)")
		}
	case types.KeyTypeGeo:
		if len(value.GeoValue) == 0 {
			vc.WriteString("(empty geo set)")
		} else {
			for _, g := range value.GeoValue {
				fmt.Fprintf(&vc, "%s  (%.6f, %.6f)\n", g.Name, g.Longitude, g.Latitude)
			}
		}
	}
	body := strings.TrimSpace(vc.String())
	if value.Truncated {
		marker := fmt.Sprintf("(truncated — %d total)", value.TotalCount)
		if body == "" {
			return marker
		}
		return body + "\n" + marker
	}
	return body
}

// formatProtobufValue renders decoded protobuf metadata and text body.
func formatProtobufValue(v types.RedisValue) string {
	var b strings.Builder
	format := v.DecodedFormat
	if format == "" {
		format = "protobuf"
	}
	fmt.Fprintf(&b, "Format: %s\n", format)
	if v.RawSize > 0 {
		if v.DecodedSize > 0 && v.DecodedSize != v.RawSize {
			fmt.Fprintf(&b, "Size: %s compressed → %s decoded\n", formatBytes(int64(v.RawSize)), formatBytes(int64(v.DecodedSize)))
		} else {
			fmt.Fprintf(&b, "Size: %s\n", formatBytes(int64(v.RawSize)))
		}
	}
	b.WriteString("\n")
	if v.DecodedValue != "" {
		b.WriteString(v.DecodedValue)
	} else {
		b.WriteString("(unable to decode protobuf payload)")
	}
	return b.String()
}

func (m Model) detailMaxScroll() int {
	maxVisible := detailMaxVisible(m.Height)
	totalLines := m.detailContentLines()
	if totalLines <= maxVisible {
		return 0
	}
	// Match scrollValueLines at scroll=0: one slot reserved for the bottom hint.
	// detailMaxVisible is always >= 5, so maxVisible-1 is always usable content.
	return totalLines - (maxVisible - 1)
}

func (m Model) viewAddKey() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Add Key"))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Type: "))
	b.WriteString(getTypeStyleBold(m.AddKeyType).Render(string(m.AddKeyType)))
	b.WriteString(dimStyle.Render(" (Ctrl+T to change)"))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Key Name:"))
	b.WriteString("\n")
	b.WriteString(m.AddKeyInputs[0].View())
	b.WriteString("\n\n")

	// Determine labels and whether to show the third input based on type
	valueLabel := "Value:"
	showThirdInput := false
	thirdLabel := ""

	switch m.AddKeyType {
	case types.KeyTypeList:
		valueLabel = "Element:"
	case types.KeyTypeSet:
		valueLabel = "Member:"
	case types.KeyTypeZSet:
		valueLabel = "Member:"
		showThirdInput = true
		thirdLabel = "Score:"
	case types.KeyTypeHash:
		valueLabel = "Field:"
		showThirdInput = true
		thirdLabel = "Value:"
	case types.KeyTypeStream:
		valueLabel = "Field:"
		showThirdInput = true
		thirdLabel = "Value:"
	case types.KeyTypeJSON:
		valueLabel = "JSON Value:"
	case types.KeyTypeHyperLogLog:
		valueLabel = "Element:"
	case types.KeyTypeBitmap:
		valueLabel = "Offset:"
	case types.KeyTypeGeo:
		valueLabel = "Member:"
		showThirdInput = true
		thirdLabel = "Lon,Lat:"
	}

	b.WriteString(keyStyle.Render(valueLabel))
	b.WriteString("\n")
	b.WriteString(m.AddKeyInputs[1].View())
	b.WriteString("\n\n")

	if showThirdInput {
		b.WriteString(keyStyle.Render(thirdLabel))
		b.WriteString("\n")
		b.WriteString(m.AddKeyInputs[2].View())
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render("tab:next  Ctrl+T:type  enter:save  esc:cancel"))

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
