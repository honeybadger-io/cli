package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	hbapi "github.com/honeybadger-io/api-go"
	"github.com/rivo/tview"
)

// StatuspagesView displays a list of status pages for an account
type StatuspagesView struct {
	app         *App
	accountID   string
	table       *tview.Table
	statuspages []hbapi.StatusPage
}

// NewStatuspagesView creates a new status pages view
func NewStatuspagesView(app *App, accountID string) *StatuspagesView {
	v := &StatuspagesView{
		app:       app,
		accountID: accountID,
		table:     tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *StatuspagesView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Status Pages ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"ID", "NAME", "URL", "SITES", "CHECK-INS"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}
}

// Name returns the view name
func (v *StatuspagesView) Name() string {
	return "Status Pages"
}

// Render returns the view's primitive
func (v *StatuspagesView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *StatuspagesView) Refresh() error {
	statuspages, err := v.app.Client().StatusPages.List(v.app.Context(), v.accountID)
	if err != nil {
		return fmt.Errorf("failed to list status pages: %w", err)
	}

	v.statuspages = statuspages
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *StatuspagesView) renderTable() {
	clearTableRows(v.table)

	if len(v.statuspages) == 0 {
		showEmptyState(v.table, "No status pages found")
		return
	}

	for i, sp := range v.statuspages {
		row := i + 1
		v.table.SetCell(row, 0, tview.NewTableCell(sp.ID).SetExpansion(1))
		v.table.SetCell(row, 1, tview.NewTableCell(sp.Name).SetExpansion(2))
		v.table.SetCell(row, 2, tview.NewTableCell(sp.URL).SetExpansion(3))
		v.table.SetCell(
			row,
			3,
			tview.NewTableCell(fmt.Sprintf("%d", len(sp.Sites))).SetExpansion(1),
		)
		v.table.SetCell(
			row,
			4,
			tview.NewTableCell(fmt.Sprintf("%d", len(sp.CheckIns))).SetExpansion(1),
		)
	}

	v.table.Select(1, 0)
	v.table.ScrollToBeginning()
}

// HandleInput handles keyboard input
func (v *StatuspagesView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}
