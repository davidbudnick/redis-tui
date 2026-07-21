package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).MarginBottom(1)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	normalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	// selectedStyle: bold dark text on accent cyan/blue band for the active detail line.
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("39"))
	keyStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	descStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	// metaDimStyle is brighter than dimStyle for Format/Size headers and scroll hints.
	metaDimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

	// Pre-allocated type-color styles to avoid per-frame allocations
	typeStyleMap = map[types.KeyType]lipgloss.Style{
		types.KeyTypeString:      lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		types.KeyTypeList:        lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		types.KeyTypeSet:         lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
		types.KeyTypeZSet:        lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		types.KeyTypeHash:        lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		types.KeyTypeStream:      lipgloss.NewStyle().Foreground(lipgloss.Color("13")),
		types.KeyTypeJSON:        lipgloss.NewStyle().Foreground(lipgloss.Color("208")),
		types.KeyTypeHyperLogLog: lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		types.KeyTypeBitmap:      lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
		types.KeyTypeGeo:         lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		types.KeyTypeProtobuf:    lipgloss.NewStyle().Foreground(lipgloss.Color("141")),
	}
	typeStyleBoldMap = map[types.KeyType]lipgloss.Style{
		types.KeyTypeString:      lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
		types.KeyTypeList:        lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true),
		types.KeyTypeSet:         lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true),
		types.KeyTypeZSet:        lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true),
		types.KeyTypeHash:        lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true),
		types.KeyTypeStream:      lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true),
		types.KeyTypeJSON:        lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true),
		types.KeyTypeHyperLogLog: lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true),
		types.KeyTypeBitmap:      lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true),
		types.KeyTypeGeo:         lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true),
		types.KeyTypeProtobuf:    lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true),
	}
	defaultTypeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	defaultTypeStyleBold = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)

	// Pre-allocated TTL warning styles
	ttlCriticalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	ttlWarningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	ttlGreenStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	// Pre-allocated hash field name style (for preview panel)
	hashFieldStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	// Pre-allocated zset score style (for preview panel)
	zsetScoreStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

func getTypeStyle(keyType types.KeyType) lipgloss.Style {
	if s, ok := typeStyleMap[keyType]; ok {
		return s
	}
	return defaultTypeStyle
}

func getTypeStyleBold(keyType types.KeyType) lipgloss.Style {
	if s, ok := typeStyleBoldMap[keyType]; ok {
		return s
	}
	return defaultTypeStyleBold
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// sanitizeBinaryString checks if a string contains binary/non-printable data
// and returns a safe representation for terminal display
func sanitizeBinaryString(s string) (string, bool) {
	// Check for common binary data signatures
	if strings.HasPrefix(s, "HYLL") { // HyperLogLog
		return fmt.Sprintf("(HyperLogLog data, %d bytes)", len(s)), true
	}

	// Count non-printable characters
	nonPrintable := 0
	for _, r := range s {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			nonPrintable++
		}
		if r > 126 && r < 160 {
			nonPrintable++
		}
	}

	// If more than 10% non-printable, treat as binary
	if len(s) > 0 && float64(nonPrintable)/float64(len(s)) > 0.1 {
		return fmt.Sprintf("(binary data, %d bytes)", len(s)), true
	}

	// Replace any remaining problematic characters
	var result strings.Builder
	for _, r := range s {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			fmt.Fprintf(&result, "\\x%02x", r)
		} else if r > 126 && r < 160 {
			fmt.Fprintf(&result, "\\x%02x", r)
		} else {
			result.WriteRune(r)
		}
	}

	return result.String(), false
}

func formatPossibleJSON(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return s
	}

	// First, check if this is binary data
	sanitized, isBinary := sanitizeBinaryString(s)
	if isBinary {
		return sanitized
	}
	s = sanitized

	// Check if it looks like JSON (starts with { or [)
	if len(s) > 0 && (s[0] == '{' || s[0] == '[') {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, []byte(s), "", "  "); err == nil {
			return colorizeJSON(prettyJSON.String())
		}
	}
	return s
}

// Shared syntax-highlight styles (jq-style).
var (
	jsonKeyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // Blue for keys / field numbers
	stringValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("34"))  // Green for strings
	jsonNumberStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))  // Yellow for numbers
	boolStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("35"))  // Magenta for booleans
	jsonNullStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("90"))  // Gray for null
	bracketStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))  // White for brackets
)

// colorizeJSON adds jq-style syntax highlighting to JSON
func colorizeJSON(s string) string {
	var result strings.Builder
	inString := false
	escaped := false
	isKey := false
	afterColon := false

	i := 0
	for i < len(s) {
		c := s[i]

		if escaped {
			result.WriteByte(c)
			escaped = false
			i++
			continue
		}

		if c == '\\' && inString {
			result.WriteByte(c)
			escaped = true
			i++
			continue
		}

		if c == '"' {
			if !inString {
				inString = true
				isKey = !afterColon && !isAfterArrayStart(s, i)
				afterColon = false
				end := findStringEnd(s, i+1)
				if end > i {
					str := s[i : end+1]
					if isKey {
						result.WriteString(jsonKeyStyle.Render(str))
					} else {
						result.WriteString(stringValueStyle.Render(str))
					}
					i = end + 1
					inString = false
					continue
				}
			}
			i++
			continue
		}

		if !inString {
			if c == ':' {
				result.WriteByte(c)
				afterColon = true
				i++
				continue
			}
			if c == ',' || c == '\n' {
				result.WriteByte(c)
				afterColon = false
				i++
				continue
			}
			if c == '{' || c == '}' || c == '[' || c == ']' {
				result.WriteString(bracketStyle.Render(string(c)))
				if c == '[' || c == '{' {
					afterColon = false
				}
				i++
				continue
			}
			if (c >= '0' && c <= '9') || c == '-' {
				end := i
				for end < len(s) && (s[end] >= '0' && s[end] <= '9' || s[end] == '.' || s[end] == '-' || s[end] == 'e' || s[end] == 'E' || s[end] == '+') {
					end++
				}
				result.WriteString(jsonNumberStyle.Render(s[i:end]))
				i = end
				afterColon = false
				continue
			}
			if strings.HasPrefix(s[i:], "true") {
				result.WriteString(boolStyle.Render("true"))
				i += 4
				afterColon = false
				continue
			}
			if strings.HasPrefix(s[i:], "false") {
				result.WriteString(boolStyle.Render("false"))
				i += 5
				afterColon = false
				continue
			}
			if strings.HasPrefix(s[i:], "null") {
				result.WriteString(jsonNullStyle.Render("null"))
				i += 4
				afterColon = false
				continue
			}
		}

		result.WriteByte(c)
		i++
	}

	return result.String()
}

// findStringEnd finds the closing quote of a JSON string
func findStringEnd(s string, start int) int {
	for i := start; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] == '"' {
			return i
		}
	}
	return -1
}

// isAfterArrayStart checks if we're inside an array (value context)
func isAfterArrayStart(s string, pos int) bool {
	for i := pos - 1; i >= 0; i-- {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			continue
		}
		if c == '[' || c == ',' {
			return isInArrayContext(s, i)
		}
		return false
	}
	return false
}

// isInArrayContext checks if position is within an array
func isInArrayContext(s string, pos int) bool {
	bracketCount := 0
	braceCount := 0
	for i := pos; i >= 0; i-- {
		c := s[i]
		switch c {
		case ']':
			bracketCount++
		case '[':
			if bracketCount > 0 {
				bracketCount--
			} else {
				return true
			}
		case '}':
			braceCount++
		case '{':
			if braceCount > 0 {
				braceCount--
			} else {
				return false
			}
		}
	}
	return false
}

// colorizeProtobuf highlights schema-less protobuf text (field numbers, strings, numbers, braces).
func colorizeProtobuf(s string) string {
	if s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = colorizeProtobufLine(line)
	}
	return strings.Join(lines, "\n")
}

// colorizeProtobufLine highlights a single protobuf decode_raw line.
func colorizeProtobufLine(line string) string {
	indentLen := 0
	for indentLen < len(line) && line[indentLen] == ' ' {
		indentLen++
	}
	indent := line[:indentLen]
	rest := line[indentLen:]
	if rest == "" {
		return line
	}

	switch {
	case strings.HasPrefix(rest, "Format:"), strings.HasPrefix(rest, "Size:"):
		return metaDimStyle.Render(line)
	case strings.HasPrefix(rest, "…"), strings.HasPrefix(rest, "↑"), strings.HasPrefix(rest, "↓"):
		return metaDimStyle.Render(line)
	case strings.HasPrefix(rest, "("):
		return metaDimStyle.Render(line)
	case rest == "{" || rest == "}":
		return indent + bracketStyle.Render(rest)
	}

	colon := strings.IndexByte(rest, ':')
	if colon <= 0 {
		return line
	}
	numPart := rest[:colon]
	for _, c := range numPart {
		if c < '0' || c > '9' {
			return line
		}
	}

	valPart := ""
	if colon+1 < len(rest) {
		if rest[colon+1] == ' ' {
			valPart = rest[colon+2:]
		} else {
			valPart = rest[colon+1:]
		}
	}

	var b strings.Builder
	b.WriteString(indent)
	b.WriteString(jsonKeyStyle.Render(numPart))
	b.WriteString(": ")
	switch {
	case valPart == "{" || valPart == "}":
		b.WriteString(bracketStyle.Render(valPart))
	case strings.HasPrefix(valPart, `"`):
		b.WriteString(stringValueStyle.Render(valPart))
	case strings.HasPrefix(valPart, "<"):
		b.WriteString(metaDimStyle.Render(valPart))
	case valPart != "":
		b.WriteString(jsonNumberStyle.Render(valPart))
	}
	return b.String()
}

// padRight pads plain text to width with trailing spaces (for full-width selection bands).
func padRight(s string, width int) string {
	n := len([]rune(s))
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

// truncateRunes truncates plain text to at most width runes.
func truncateRunes(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

// wrapPlainLines hard-wraps lines to width so scroll counts match the rendered box.
func wrapPlainLines(lines []string, width int) []string {
	if width < 8 {
		width = 8
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			out = append(out, "")
			continue
		}
		runes := []rune(line)
		for len(runes) > width {
			out = append(out, string(runes[:width]))
			runes = runes[width:]
		}
		out = append(out, string(runes))
	}
	return out
}

// detailBoxWidth is the centered detail value box width for the current terminal size.
func detailBoxWidth(width int) int {
	boxWidth := width * 3 / 5
	boxWidth = max(boxWidth, 50)
	boxWidth = min(boxWidth, width-6)
	return boxWidth
}

// detailContentWidth is the usable text width inside the value box (excludes padding).
func detailContentWidth(boxWidth int) int {
	w := boxWidth - 4 // Padding(1, 2) left+right
	if w < 20 {
		return 20
	}
	return w
}

// detailChromeLines is vertical space used by title, meta, borders, padding, and help.
const detailChromeLines = 16

// detailMaxVisible returns how many content lines fit in the value box.
func detailMaxVisible(height int) int {
	maxVisible := height - detailChromeLines
	if maxVisible < 5 {
		return 5
	}
	return maxVisible
}

// scrollValueLines windows lines into maxVisible slots, reserving space for scroll hints.
func scrollValueLines(valueLines []string, scroll, maxVisible int) (visible []string, topHint, bottomHint string, clampedScroll int) {
	if maxVisible < 1 {
		maxVisible = 1
	}
	clampedScroll = max(scroll, 0)
	total := len(valueLines)
	if total <= maxVisible {
		return valueLines, "", "", 0
	}

	// Content rows available after reserving hint rows inside the box.
	// While not at EOF we always reserve 1 for "↓ more"; when scrolled, also 1 for "↑ more".
	contentRows := func(scrolled bool) int {
		n := maxVisible - 1
		if scrolled {
			n--
		}
		return max(n, 1)
	}

	avail := contentRows(clampedScroll > 0)
	maxScroll := total - avail
	clampedScroll = min(clampedScroll, maxScroll)
	avail = contentRows(clampedScroll > 0)
	end := min(clampedScroll+avail, total)

	// At EOF drop the bottom hint and fill with content.
	if end >= total {
		rows := maxVisible
		if clampedScroll > 0 {
			rows--
		}
		end = min(clampedScroll+max(rows, 1), total)
	}

	visible = valueLines[clampedScroll:end]
	if clampedScroll > 0 {
		topHint = metaDimStyle.Render(fmt.Sprintf("↑ %d more lines above", clampedScroll))
	}
	if end < total {
		bottomHint = metaDimStyle.Render(fmt.Sprintf("↓ %d more lines below", total-end))
	}
	return visible, topHint, bottomHint, clampedScroll
}

// ensureDetailCursorVisible adjusts DetailScroll so DetailCursor stays in the viewport.
func ensureDetailCursorVisible(cursor, scroll, total, maxVisible int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}
	// Worst-case content rows when both scroll hints are shown.
	window := maxVisible - 2
	if window < 1 {
		window = 1
	}
	if cursor < scroll {
		scroll = cursor
	}
	if cursor >= scroll+window {
		scroll = cursor - window + 1
	}
	if scroll < 0 {
		scroll = 0
	}
	return cursor, scroll
}

func (m Model) renderModal(content string) string {
	modalWidth := min(60, m.Width-10)
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(modalWidth)

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, modalStyle.Render(content))
}

func (m Model) renderModalWide(content string) string {
	modalWidth := min(90, m.Width-10)
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(modalWidth)

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, modalStyle.Render(content))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

type logEntry struct {
	Time  string
	Level string
	Msg   string
}

func parseLogEntry(logLine string) logEntry {
	entry := logEntry{}

	var data map[string]any
	if err := json.Unmarshal([]byte(logLine), &data); err != nil {
		entry.Msg = logLine
		return entry
	}

	if t, ok := data["time"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339Nano, t); err == nil {
			entry.Time = parsed.Format("15:04:05")
		} else {
			entry.Time = t
		}
	}
	if l, ok := data["level"].(string); ok {
		entry.Level = strings.ToUpper(l)
	}
	if m, ok := data["msg"].(string); ok {
		entry.Msg = m
	}

	return entry
}
