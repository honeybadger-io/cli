package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	hbapi "github.com/honeybadger-io/api-go"
	"github.com/rivo/tview"
)

// CheckinsView displays check-ins for a project
type CheckinsView struct {
	app       *App
	projectID int
	table     *tview.Table
	checkins  []hbapi.CheckIn
}

// NewCheckinsView creates a new check-ins view
func NewCheckinsView(app *App, projectID int) *CheckinsView {
	v := &CheckinsView{
		app:       app,
		projectID: projectID,
		table:     tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *CheckinsView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Check-ins ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"ID", "NAME", "SLUG", "TYPE", "SCHEDULE", "LAST CHECK-IN"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}

	v.table.SetSelectedFunc(func(row, col int) {
		if row > 0 && row <= len(v.checkins) {
			checkin := v.checkins[row-1]
			v.showDetails(checkin)
		}
	})
}

func (v *CheckinsView) showDetails(checkin hbapi.CheckIn) {
	detailsView := NewCheckinDetailsView(v.app, checkin)
	v.app.Push(detailsView)
}

// Name returns the view name
func (v *CheckinsView) Name() string {
	return "Check-ins"
}

// Render returns the view's primitive
func (v *CheckinsView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *CheckinsView) Refresh() error {
	checkins, err := v.app.Client().CheckIns.List(v.app.Context(), v.projectID)
	if err != nil {
		return fmt.Errorf("failed to list check-ins: %w", err)
	}

	v.checkins = checkins
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *CheckinsView) renderTable() {
	for row := v.table.GetRowCount() - 1; row > 0; row-- {
		v.table.RemoveRow(row)
	}

	for i, ci := range v.checkins {
		row := i + 1

		schedule := ""
		if ci.ScheduleType == "simple" && ci.ReportPeriod != nil {
			schedule = *ci.ReportPeriod
		} else if ci.ScheduleType == "cron" && ci.CronSchedule != nil {
			schedule = *ci.CronSchedule
		}

		lastCheckIn := "Never"
		lastCheckInColor := tcell.ColorYellow
		if ci.LastCheckInAt != nil {
			lastCheckIn = ci.LastCheckInAt.Format("2006-01-02 15:04")
			lastCheckInColor = tcell.ColorGreen
		}

		v.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", ci.ID)).SetExpansion(1))
		v.table.SetCell(row, 1, tview.NewTableCell(ci.Name).SetExpansion(2))
		v.table.SetCell(row, 2, tview.NewTableCell(ci.Slug).SetExpansion(2))
		v.table.SetCell(row, 3, tview.NewTableCell(ci.ScheduleType).SetExpansion(1))
		v.table.SetCell(row, 4, tview.NewTableCell(schedule).SetExpansion(2))
		v.table.SetCell(row, 5, tview.NewTableCell(lastCheckIn).SetTextColor(lastCheckInColor).SetExpansion(2))
	}

	if len(v.checkins) > 0 {
		v.table.Select(1, 0)
	}
}

// HandleInput handles keyboard input
func (v *CheckinsView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Rune() {
	case 'j':
		row, col := v.table.GetSelection()
		if row < v.table.GetRowCount()-1 {
			v.table.Select(row+1, col)
		}
		return nil
	case 'k':
		row, col := v.table.GetSelection()
		if row > 1 {
			v.table.Select(row-1, col)
		}
		return nil
	case 'l':
		row, _ := v.table.GetSelection()
		if row > 0 && row <= len(v.checkins) {
			checkin := v.checkins[row-1]
			v.showDetails(checkin)
		}
		return nil
	case 'h':
		v.app.Pop()
		return nil
	}

	if event.Key() == tcell.KeyEnter || event.Key() == tcell.KeyRight {
		row, _ := v.table.GetSelection()
		if row > 0 && row <= len(v.checkins) {
			checkin := v.checkins[row-1]
			v.showDetails(checkin)
		}
		return nil
	}

	return event
}

// CheckinDetailsView displays detailed information about a check-in
type CheckinDetailsView struct {
	app      *App
	checkin  hbapi.CheckIn
	textView *tview.TextView
}

// NewCheckinDetailsView creates a new check-in details view
func NewCheckinDetailsView(app *App, checkin hbapi.CheckIn) *CheckinDetailsView {
	v := &CheckinDetailsView{
		app:      app,
		checkin:  checkin,
		textView: tview.NewTextView(),
	}
	v.setupView()
	return v
}

func (v *CheckinDetailsView) setupView() {
	v.textView.SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)

	v.textView.SetTitle(" Check-in Details ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)
}

// Name returns the view name
func (v *CheckinDetailsView) Name() string {
	return "Details"
}

// Render returns the view's primitive
func (v *CheckinDetailsView) Render() tview.Primitive {
	return v.textView
}

// Refresh reloads the data
func (v *CheckinDetailsView) Refresh() error {
	v.app.Application().QueueUpdateDraw(func() {
		v.renderDetails()
	})
	return nil
}

func (v *CheckinDetailsView) renderDetails() {
	ci := v.checkin

	text := fmt.Sprintf(`[yellow]ID:[white] %d

[yellow]Name:[white] %s

[yellow]Slug:[white] %s

[yellow]Schedule Type:[white] %s`,
		ci.ID,
		ci.Name,
		ci.Slug,
		ci.ScheduleType,
	)

	if ci.ReportPeriod != nil {
		text += fmt.Sprintf("\n\n[yellow]Report Period:[white] %s", *ci.ReportPeriod)
	}

	if ci.GracePeriod != nil {
		text += fmt.Sprintf("\n[yellow]Grace Period:[white] %s", *ci.GracePeriod)
	}

	if ci.CronSchedule != nil {
		text += fmt.Sprintf("\n\n[yellow]Cron Schedule:[white] %s", *ci.CronSchedule)
	}

	if ci.CronTimezone != nil {
		text += fmt.Sprintf("\n[yellow]Cron Timezone:[white] %s", *ci.CronTimezone)
	}

	text += fmt.Sprintf("\n\n[yellow]Project ID:[white] %d", ci.ProjectID)
	text += fmt.Sprintf("\n[yellow]Created:[white] %s", ci.CreatedAt.Format("2006-01-02 15:04:05"))

	if ci.LastCheckInAt != nil {
		text += fmt.Sprintf("\n\n[yellow]Last Check-in:[green] %s", ci.LastCheckInAt.Format("2006-01-02 15:04:05"))
	} else {
		text += "\n\n[yellow]Last Check-in:[red] Never"
	}

	v.textView.SetText(text)
}

// HandleInput handles keyboard input
func (v *CheckinDetailsView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Rune() {
	case 'h':
		v.app.Pop()
		return nil
	}
	return event
}
