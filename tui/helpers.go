package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// clearTableRows removes all rows from a table except the header row (row 0)
func clearTableRows(table *tview.Table) {
	for row := table.GetRowCount() - 1; row > 0; row-- {
		table.RemoveRow(row)
	}
}

// truncateString safely truncates a string to maxLen characters, UTF-8 aware.
// If the string is longer than maxLen, it appends "..." and ensures the total
// length does not exceed maxLen.
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return string(runes[:maxLen])
	}

	return string(runes[:maxLen-3]) + "..."
}

// handleTableNavigation handles common j/k navigation for tables.
// Returns true if the event was handled, false otherwise.
func handleTableNavigation(table *tview.Table, event *tcell.EventKey) bool {
	switch event.Rune() {
	case 'j':
		row, col := table.GetSelection()
		if row < table.GetRowCount()-1 {
			table.Select(row+1, col)
		}
		return true
	case 'k':
		row, col := table.GetSelection()
		if row > 1 { // Row 0 is header
			table.Select(row-1, col)
		}
		return true
	}
	return false
}

// handleListNavigation handles common j/k navigation for lists.
// Returns true if the event was handled, false otherwise.
func handleListNavigation(list *tview.List, event *tcell.EventKey) bool {
	switch event.Rune() {
	case 'j':
		currentItem := list.GetCurrentItem()
		if currentItem < list.GetItemCount()-1 {
			list.SetCurrentItem(currentItem + 1)
		}
		return true
	case 'k':
		currentItem := list.GetCurrentItem()
		if currentItem > 0 {
			list.SetCurrentItem(currentItem - 1)
		}
		return true
	}
	return false
}

// handleBackNavigation handles h key for going back.
// Returns true if the event was handled, false otherwise.
func handleBackNavigation(app *App, event *tcell.EventKey) bool {
	if event.Rune() == 'h' {
		app.Pop()
		return true
	}
	return false
}

// isSelectKey returns true if the event is a selection key (Enter, Right, or 'l')
func isSelectKey(event *tcell.EventKey) bool {
	return event.Key() == tcell.KeyEnter ||
		event.Key() == tcell.KeyRight ||
		event.Rune() == 'l'
}

// showEmptyState adds a "No results" message to an empty table
func showEmptyState(table *tview.Table, message string) {
	cell := tview.NewTableCell(message).
		SetTextColor(tcell.ColorGray).
		SetSelectable(false).
		SetExpansion(1)
	table.SetCell(1, 0, cell)
}
