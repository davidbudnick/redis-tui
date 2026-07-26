package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) viewHelp() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Help"))
	b.WriteString("\n\n")

	sections := []struct {
		title    string
		bindings [][2]string
	}{
		{
			title: "Global",
			bindings: [][2]string{
				{"q", "Quit / Go back"},
				{"?", "Show help"},
				{"j/k", "Navigate up/down"},
				{"Ctrl+U/D", "Page up/down"},
				{"g/G", "Top/Bottom"},
				{"Ctrl+Y", "Copy input text"},
			},
		},
		{
			title: "Connections",
			bindings: [][2]string{
				{"a/n", "Add connection"},
				{"e", "Edit connection"},
				{"d", "Delete connection"},
				{"r", "Refresh list"},
				{"Ctrl+T", "Test connection"},
			},
		},
		{
			title: "Keys",
			bindings: [][2]string{
				{"enter", "View key detail"},
				{"a/n", "Add key"},
				{"d", "Delete key"},
				{"r", "Refresh keys"},
				{"l", "Load more keys"},
				{"/", "Filter by pattern"},
				{"y", "Copy key name"},
				{"s/S", "Sort / Toggle direction"},
				{"v", "Search by value"},
				{"e", "Export to JSON"},
				{"I", "Import from JSON"},
				{"D", "Switch database"},
				{"i", "Server info"},
				{"f", "Flush database"},
				{"p", "Pub/Sub publish"},
				{"L", "View slow log"},
				{"E", "Execute Lua script"},
				{"O", "View application logs"},
				{"B", "Bulk delete"},
				{"T", "Batch set TTL"},
				{"F", "View favorites"},
				{"W", "Tree view"},
				{"Ctrl+R", "Regex search"},
				{"Ctrl+F", "Fuzzy search"},
				{"Ctrl+H", "Recent keys"},
				{"Ctrl+L", "Client list"},
				{"Ctrl+E", "Keyspace events"},
				{"Ctrl+X", "Expiring keys"},
				{"m", "Live metrics"},
				{"M", "Memory stats"},
				{"C", "Cluster info"},
				{"K", "Compare keys"},
				{"P", "Key templates"},
			},
		},
		{
			title: "Key Detail",
			bindings: [][2]string{
				{"e", "Edit value (string/json)"},
				{"a", "Add to collection"},
				{"x", "Remove from collection"},
				{"R", "Rename key"},
				{"c", "Copy key"},
				{"d", "Delete key"},
				{"t", "Set TTL"},
				{"r", "Refresh"},
				{"f", "Toggle favorite"},
				{"h", "Value history"},
				{"y", "Copy to clipboard"},
				{"J", "JSON path query"},
			},
		},
	}

	colStyle := lipgloss.NewStyle().Width(34)

	for _, section := range sections {
		b.WriteString(keyStyle.Render(section.title))
		b.WriteString("\n")
		half := (len(section.bindings) + 1) / 2
		var leftCol, rightCol strings.Builder
		for i, binding := range section.bindings {
			line := fmt.Sprintf("  %s %s", dimStyle.Render(binding[0]), descStyle.Render(binding[1]))
			if i < half {
				leftCol.WriteString(line + "\n")
			} else {
				rightCol.WriteString(line + "\n")
			}
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, colStyle.Render(leftCol.String()), colStyle.Render(rightCol.String())))
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("Press ? or esc to close"))

	modalWidth := 80
	if m.Width-10 < 80 {
		modalWidth = m.Width - 10
	}
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(modalWidth)

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, modalStyle.Render(b.String()))
}

func (m Model) viewTestConnection() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Test Connection"))
	b.WriteString("\n\n")

	if m.Loading {
		b.WriteString(dimStyle.Render("Testing connection..."))
	} else if m.TestConnResult != "" {
		if strings.HasPrefix(m.TestConnResult, "Failed") {
			b.WriteString(errorStyle.Render(m.TestConnResult))
		} else {
			b.WriteString(successStyle.Render(m.TestConnResult))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("esc:back"))

	return m.renderModal(b.String())
}
