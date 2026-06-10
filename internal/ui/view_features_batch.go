package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) viewBulkDelete() string {
	var b strings.Builder

	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)

	b.WriteString(warningStyle.Render("Bulk Delete Keys"))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("Delete all keys matching a pattern"))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Pattern:"))
	b.WriteString("\n")
	b.WriteString(m.Inputs.BulkDeleteInput.View())
	b.WriteString("\n\n")

	if len(m.BulkDeletePreview) > 0 {
		b.WriteString(keyStyle.Render(fmt.Sprintf("Will delete %d keys:", len(m.BulkDeletePreview))))
		b.WriteString("\n")
		for i, k := range m.BulkDeletePreview {
			if i >= 5 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more", len(m.BulkDeletePreview)-5)))
				break
			}
			b.WriteString(normalStyle.Render("  • " + k))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("enter:delete  esc:cancel"))

	return m.renderModal(b.String())
}

func (m Model) viewBatchTTL() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Batch Set TTL"))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("Set TTL on all keys matching a pattern"))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("TTL (seconds):"))
	b.WriteString("\n")
	b.WriteString(m.Inputs.BatchTTLInput.View())
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Pattern:"))
	b.WriteString("\n")
	b.WriteString(m.Inputs.BatchTTLPattern.View())
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("tab:next  enter:apply  esc:cancel"))

	return m.renderModal(b.String())
}
