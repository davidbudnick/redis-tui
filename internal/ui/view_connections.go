package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

var (
	// Logo style
	logoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	// Accent colors
	accentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	// Stats box style
	statsBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	// Connection card style
	connCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			MarginBottom(0)

	connCardSelectedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("39")).
				Padding(0, 1).
				MarginBottom(0)
)

func (m Model) viewConnections() string {
	var b strings.Builder

	// ASCII Art Logo
	logo := `
 ██████╗ ███████╗██████╗ ██╗███████╗
 ██╔══██╗██╔════╝██╔══██╗██║██╔════╝
 ██████╔╝█████╗  ██║  ██║██║███████╗
 ██╔══██╗██╔══╝  ██║  ██║██║╚════██║
 ██║  ██║███████╗██████╔╝██║███████║
 ╚═╝  ╚═╝╚══════╝╚═════╝ ╚═╝╚══════╝`

	b.WriteString(logoStyle.Render(logo))
	b.WriteString("\n\n")

	// Stats bar
	statsContent := m.buildStatsBar()
	b.WriteString(statsContent)
	b.WriteString("\n\n")

	// Connection error display (prominent error box)
	if m.ConnectionError != "" {
		errorBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Foreground(lipgloss.Color("196")).
			Padding(0, 2).
			Width(55).
			Render(fmt.Sprintf("Connection Failed\n%s", dimStyle.Render(m.ConnectionError)))
		b.WriteString(errorBox)
		b.WriteString("\n\n")
	}

	// Section title
	connCount := len(m.Connections)
	sectionTitle := fmt.Sprintf("╭─ Saved Connections (%d) ", connCount)
	sectionTitle += strings.Repeat("─", 50-len(sectionTitle)) + "╮"
	b.WriteString(accentStyle.Render(sectionTitle))
	b.WriteString("\n")

	if len(m.Connections) == 0 {
		b.WriteString("\n")
		emptyBox := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(1, 2).
			Render("  No connections saved.\n\n  Press 'a' to add your first Redis connection.")
		b.WriteString(emptyBox)
		b.WriteString("\n")
	} else {
		b.WriteString("\n")

		// Calculate visible range for scrolling
		maxVisible := max((m.Height-20)/3, 3)

		// Ensure selected index is within bounds
		selectedIdx := m.SelectedConnIdx
		if selectedIdx >= len(m.Connections) {
			selectedIdx = len(m.Connections) - 1
		}
		if selectedIdx < 0 {
			selectedIdx = 0
		}

		startIdx := 0
		if selectedIdx >= maxVisible {
			startIdx = selectedIdx - maxVisible + 1
		}
		endIdx := startIdx + maxVisible
		if endIdx > len(m.Connections) {
			endIdx = len(m.Connections)
			// Adjust startIdx to show more items when at end of list
			if endIdx-startIdx < maxVisible {
				startIdx = max(endIdx-maxVisible, 0)
			}
		}

		for i := startIdx; i < endIdx; i++ {
			conn := m.Connections[i]
			isSelected := i == selectedIdx

			// Build connection card content
			var card strings.Builder

			// Connection name with icon
			icon := "○"
			if isSelected {
				icon = "●"
			}

			nameStyle := normalStyle
			if isSelected {
				nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
			}
			fmt.Fprintf(&card, " %s %s", icon, nameStyle.Render(conn.Name))
			card.WriteString("\n")

			// Connection details
			hostPort := fmt.Sprintf("   %s:%d", conn.Host, conn.Port)
			card.WriteString(dimStyle.Render(hostPort))

			// Database badge (hide in cluster mode — clusters don't use databases)
			if !conn.UseCluster {
				dbBadge := lipgloss.NewStyle().
					Background(lipgloss.Color("236")).
					Foreground(lipgloss.Color("245")).
					Padding(0, 1).
					Render(fmt.Sprintf("db%d", conn.DB))
				card.WriteString("  ")
				card.WriteString(dbBadge)
			}

			// TLS indicator
			if conn.UseTLS {
				tlsBadge := lipgloss.NewStyle().
					Background(lipgloss.Color("22")).
					Foreground(lipgloss.Color("46")).
					Padding(0, 1).
					Render("TLS")
				card.WriteString(" ")
				card.WriteString(tlsBadge)
			}

			// Cluster indicator
			if conn.UseCluster {
				clusterBadge := lipgloss.NewStyle().
					Background(lipgloss.Color("53")).
					Foreground(lipgloss.Color("213")).
					Padding(0, 1).
					Render("CLUSTER")
				card.WriteString(" ")
				card.WriteString(clusterBadge)
			}

			// Render the card with appropriate style
			cardStyle := connCardStyle
			if isSelected {
				cardStyle = connCardSelectedStyle
			}

			// Set card width
			cardWidth := min(55, m.Width-10)
			cardStyle = cardStyle.Width(cardWidth)

			b.WriteString(cardStyle.Render(card.String()))
			b.WriteString("\n")
		}

		// Scroll indicator
		if len(m.Connections) > maxVisible {
			scrollInfo := fmt.Sprintf("  ↕ %d-%d of %d connections", startIdx+1, endIdx, len(m.Connections))
			b.WriteString(dimStyle.Render(scrollInfo))
			b.WriteString("\n")
		}
	}

	// Bottom section line
	sectionBottom := "╰" + strings.Repeat("─", 54) + "╯"
	b.WriteString(accentStyle.Render(sectionBottom))
	b.WriteString("\n\n")

	// Keybindings footer
	keybindings := []struct {
		key  string
		desc string
	}{
		{"↑/↓", "navigate"},
		{"enter", "connect"},
		{"a", "add"},
		{"e", "edit"},
		{"d", "delete"},
		{"q", "quit"},
	}

	var keyHelp strings.Builder
	for i, kb := range keybindings {
		keyStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1).
			Render(kb.key)
		keyHelp.WriteString(keyStyle)
		keyHelp.WriteString(" ")
		keyHelp.WriteString(dimStyle.Render(kb.desc))
		if i < len(keybindings)-1 {
			keyHelp.WriteString("  ")
		}
	}
	b.WriteString(keyHelp.String())

	return b.String()
}

func (m Model) buildStatsBar() string {
	// Create stats boxes
	boxes := []struct {
		label string
		value string
		color string
	}{
		{"Connections", fmt.Sprintf("%d saved", len(m.Connections)), "39"},
		{"Time", time.Now().Format("15:04:05"), "245"},
	}

	var statsBoxes []string
	for _, box := range boxes {
		content := fmt.Sprintf("%s\n%s",
			dimStyle.Render(box.label),
			lipgloss.NewStyle().Foreground(lipgloss.Color(box.color)).Bold(true).Render(box.value),
		)
		styled := statsBoxStyle.Width(18).Render(content)
		statsBoxes = append(statsBoxes, styled)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, statsBoxes...)
}

func (m Model) viewAddConnection() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Add Connection"))
	b.WriteString("\n\n")

	b.WriteString(m.renderConnForm())

	// Action buttons hint
	actions := lipgloss.NewStyle().
		Background(lipgloss.Color("22")).
		Foreground(lipgloss.Color("46")).
		Padding(0, 1).
		Render("Ctrl+T: Test")
	b.WriteString(actions)
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("tab:next  space:toggle  enter:save  esc:cancel"))

	modalWidth := min(55, m.Width-10)
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(modalWidth)

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, modalStyle.Render(b.String()))
}

func (m Model) viewEditConnection() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Edit Connection"))
	b.WriteString("\n\n")

	b.WriteString(m.renderConnForm())

	b.WriteString(helpStyle.Render("tab:next  space:toggle  enter:save  esc:cancel"))

	modalWidth := min(55, m.Width-10)
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(modalWidth)

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, modalStyle.Render(b.String()))
}

// renderConnForm renders the shared connection form fields.
func (m Model) renderConnForm() string {
	var b strings.Builder

	// Fields 0-7: connection and credential inputs.
	textLabels := []string{"Name", "Host", "Port", "Username", "Password", "Vault Path", "Vault Username Key", "Vault Password Key"}
	for i := range textLabels {
		labelStyle := keyStyle
		if m.ConnFocusIdx == i {
			labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
		}
		b.WriteString(labelStyle.Render(textLabels[i] + ":"))
		b.WriteString("\n")
		b.WriteString(m.ConnInputs[i].View())
		b.WriteString("\n\n")
	}

	// Field 8: Cluster toggle
	clusterLabelStyle := keyStyle
	if m.ConnFocusIdx == len(textLabels) {
		clusterLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	}
	b.WriteString(clusterLabelStyle.Render("Cluster:"))
	b.WriteString("\n")
	checkbox := "[ ] Cluster Mode"
	if m.ConnClusterMode {
		checkbox = "[x] Cluster Mode"
	}
	checkboxStyle := normalStyle
	if m.ConnFocusIdx == len(textLabels) {
		checkboxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	}
	b.WriteString(checkboxStyle.Render(checkbox))
	b.WriteString("\n\n")

	// Field 9: Database (only when not in cluster mode)
	if !m.ConnClusterMode {
		dbLabelStyle := keyStyle
		if m.ConnFocusIdx == len(textLabels)+1 {
			dbLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
		}
		b.WriteString(dbLabelStyle.Render("Database:"))
		b.WriteString("\n")
		b.WriteString(m.ConnInputs[8].View())
		b.WriteString("\n\n")
	}

	return b.String()
}
