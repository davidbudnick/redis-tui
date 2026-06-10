package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

func (m Model) viewFavorites() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Favorites"))
	b.WriteString("\n\n")

	if len(m.Favorites) == 0 {
		b.WriteString(dimStyle.Render("No favorites yet."))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Press 'f' on a key to add it."))
	} else {
		for i, fav := range m.Favorites {
			prefix := "  "
			connName := fav.Connection
			if connName == "" {
				connName = fav.Label
			}
			if i == m.SelectedFavIdx {
				b.WriteString(selectedStyle.Render(fmt.Sprintf("▶ %-40s %s", truncate(fav.Key, 40), connName)))
			} else {
				b.WriteString(normalStyle.Render(fmt.Sprintf("%s%-40s %s", prefix, truncate(fav.Key, 40), dimStyle.Render(connName))))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  enter:view  d:remove  esc:back"))

	return m.renderModal(b.String())
}

func (m Model) viewRecentKeys() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Recent Keys"))
	b.WriteString("\n\n")

	if len(m.RecentKeys) == 0 {
		b.WriteString(dimStyle.Render("No recent keys."))
	} else {
		for i, recent := range m.RecentKeys {
			if i == m.SelectedRecentIdx {
				b.WriteString(selectedStyle.Render(fmt.Sprintf("▶ %-40s", truncate(recent.Key, 40))))
			} else {
				b.WriteString(normalStyle.Render(fmt.Sprintf("  %-40s", truncate(recent.Key, 40))))
			}
			b.WriteString(dimStyle.Render(" " + recent.AccessedAt.Format("15:04:05")))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  enter:view  esc:back"))

	return m.renderModal(b.String())
}

func (m Model) viewTreeView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Key Tree View"))
	b.WriteString("\n\n")

	if len(m.TreeNodes) == 0 {
		b.WriteString(dimStyle.Render("No keys found."))
	} else {
		for i, node := range m.TreeNodes {
			indent := strings.Repeat("  ", node.GetDepth())
			prefix := "  "
			icon := "."
			if !node.IsKey {
				if m.TreeExpanded[node.FullPath] {
					icon = "-"
				} else {
					icon = "+"
				}
			}

			line := fmt.Sprintf("%s%s %s (%d)", indent, icon, node.Name, node.ChildCount)
			if i == m.SelectedTreeIdx {
				b.WriteString(selectedStyle.Render("▶ " + line[2:]))
			} else {
				b.WriteString(normalStyle.Render(prefix + line[2:]))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  enter/space:expand  esc:back"))

	return m.renderModalWide(b.String())
}

func (m Model) viewTemplates() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Key Templates"))
	b.WriteString("\n\n")

	if len(m.Templates) == 0 {
		b.WriteString(dimStyle.Render("No templates configured."))
	} else {
		for i, tmpl := range m.Templates {
			typePart := getTypeStyle(tmpl.KeyType).Render(string(tmpl.KeyType))

			if i == m.SelectedTemplateIdx {
				b.WriteString(selectedStyle.Render(fmt.Sprintf("▶ %-25s %-10s", tmpl.Name, tmpl.KeyType)))
			} else {
				b.WriteString(normalStyle.Render(fmt.Sprintf("  %-25s ", tmpl.Name)))
				b.WriteString(typePart)
			}
			b.WriteString("\n")
			b.WriteString(dimStyle.Render(fmt.Sprintf("    %s", tmpl.Pattern)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  enter:use  esc:back"))

	return m.renderModal(b.String())
}

func (m Model) viewValueHistory() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Value History"))
	b.WriteString("\n\n")

	if m.CurrentKey != nil {
		b.WriteString(keyStyle.Render("Key: "))
		b.WriteString(normalStyle.Render(m.CurrentKey.Key))
		b.WriteString("\n\n")
	}

	if len(m.ValueHistory) == 0 {
		b.WriteString(dimStyle.Render("No history available for this key."))
	} else {
		for i, entry := range m.ValueHistory {
			timeStr := entry.Timestamp.Format("2006-01-02 15:04:05")
			value := truncate(entry.Value.StringValue, 40)

			if i == m.SelectedHistoryIdx {
				b.WriteString(selectedStyle.Render(fmt.Sprintf("▶ %s  %s", timeStr, value)))
			} else {
				b.WriteString(dimStyle.Render(timeStr))
				b.WriteString("  ")
				b.WriteString(normalStyle.Render(value))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  enter:restore  esc:back"))

	return m.renderModalWide(b.String())
}

func (m Model) viewExpiringKeys() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Expiring Keys"))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render(fmt.Sprintf("Keys expiring within %d seconds", m.ExpiryThreshold)))
	b.WriteString("\n\n")

	if len(m.ExpiringKeys) == 0 {
		b.WriteString(dimStyle.Render("No keys expiring soon."))
	} else {
		header := fmt.Sprintf("  %-40s %-15s", "Key", "TTL")
		b.WriteString(headerStyle.Render(header))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(strings.Repeat("─", 60)))
		b.WriteString("\n")

		for i, key := range m.ExpiringKeys {
			keyName := truncate(key.Key, 40)
			ttlStr := key.TTL.String()

			var ttlStyle lipgloss.Style
			if key.TTL <= 10*time.Second {
				ttlStyle = errorStyle
			} else if key.TTL <= 60*time.Second {
				ttlStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
			} else {
				ttlStyle = normalStyle
			}

			if i == m.SelectedKeyIdx {
				b.WriteString(selectedStyle.Render(fmt.Sprintf("▶ %-40s %-15s", keyName, ttlStr)))
			} else {
				b.WriteString(normalStyle.Render(fmt.Sprintf("  %-40s ", keyName)))
				b.WriteString(ttlStyle.Render(ttlStr))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  enter:view  esc:back"))

	return m.renderModalWide(b.String())
}
