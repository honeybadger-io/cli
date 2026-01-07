package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	hbapi "github.com/honeybadger-io/api-go"
	"github.com/rivo/tview"
)

// FaultsView displays a list of faults for a project
type FaultsView struct {
	app       *App
	projectID int
	table     *tview.Table
	faults    []hbapi.Fault
}

// NewFaultsView creates a new faults view
func NewFaultsView(app *App, projectID int) *FaultsView {
	v := &FaultsView{
		app:       app,
		projectID: projectID,
		table:     tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *FaultsView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Faults ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"ID", "CLASS", "MESSAGE", "ENV", "NOTICES", "STATUS", "LAST SEEN"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}

	v.table.SetSelectedFunc(func(row, col int) {
		if row > 0 && row <= len(v.faults) {
			fault := v.faults[row-1]
			v.drillDown(fault)
		}
	})
}

func (v *FaultsView) drillDown(fault hbapi.Fault) {
	menuView := NewFaultMenuView(v.app, v.projectID, fault)
	v.app.Push(menuView)
}

// Name returns the view name
func (v *FaultsView) Name() string {
	return "Faults"
}

// Render returns the view's primitive
func (v *FaultsView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *FaultsView) Refresh() error {
	response, err := v.app.Client().Faults.List(v.app.Context(), v.projectID, hbapi.FaultListOptions{
		Limit: 25,
		Order: "recent",
	})
	if err != nil {
		return fmt.Errorf("failed to list faults: %w", err)
	}

	v.faults = response.Results
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *FaultsView) renderTable() {
	clearTableRows(v.table)

	for i, fault := range v.faults {
		row := i + 1

		message := truncateString(fault.Message, 40)

		status := "Active"
		statusColor := tcell.ColorRed
		if fault.Resolved {
			status = "Resolved"
			statusColor = tcell.ColorGreen
		} else if fault.Ignored {
			status = "Ignored"
			statusColor = tcell.ColorYellow
		}

		lastSeen := "-"
		if fault.LastNoticeAt != nil {
			lastSeen = fault.LastNoticeAt.Format("2006-01-02 15:04")
		}

		v.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", fault.ID)).SetExpansion(1))
		v.table.SetCell(row, 1, tview.NewTableCell(fault.Klass).SetExpansion(2))
		v.table.SetCell(row, 2, tview.NewTableCell(message).SetExpansion(3))
		v.table.SetCell(row, 3, tview.NewTableCell(fault.Environment).SetExpansion(1))
		v.table.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%d", fault.NoticesCount)).SetExpansion(1))
		v.table.SetCell(row, 5, tview.NewTableCell(status).SetTextColor(statusColor).SetExpansion(1))
		v.table.SetCell(row, 6, tview.NewTableCell(lastSeen).SetExpansion(2))
	}

	if len(v.faults) > 0 {
		v.table.Select(1, 0)
	}
}

// HandleInput handles keyboard input
func (v *FaultsView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	if isSelectKey(event) {
		row, _ := v.table.GetSelection()
		if row > 0 && row <= len(v.faults) {
			v.drillDown(v.faults[row-1])
		}
		return nil
	}
	return event
}

// FaultMenuView shows options for a selected fault
type FaultMenuView struct {
	app       *App
	projectID int
	fault     hbapi.Fault
	list      *tview.List
}

// NewFaultMenuView creates a new fault menu view
func NewFaultMenuView(app *App, projectID int, fault hbapi.Fault) *FaultMenuView {
	v := &FaultMenuView{
		app:       app,
		projectID: projectID,
		fault:     fault,
		list:      tview.NewList(),
	}
	v.setupList()
	return v
}

func (v *FaultMenuView) setupList() {
	title := truncateString(v.fault.Klass, 50)
	v.list.SetTitle(fmt.Sprintf(" %s ", title)).
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	v.list.AddItem("Details", "View fault details", 'd', func() {
		detailsView := NewFaultDetailsView(v.app, v.projectID, v.fault.ID)
		v.app.Push(detailsView)
	})

	v.list.AddItem("Notices", fmt.Sprintf("View notices (%d total)", v.fault.NoticesCount), 'n', func() {
		noticesView := NewNoticesView(v.app, v.projectID, v.fault.ID)
		v.app.Push(noticesView)
	})

	v.list.AddItem("Affected Users", "View affected users", 'u', func() {
		usersView := NewAffectedUsersView(v.app, v.projectID, v.fault.ID)
		v.app.Push(usersView)
	})

	v.list.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
}

// Name returns the view name
func (v *FaultMenuView) Name() string {
	return truncateString(v.fault.Klass, 30)
}

// Render returns the view's primitive
func (v *FaultMenuView) Render() tview.Primitive {
	return v.list
}

// Refresh reloads the data
func (v *FaultMenuView) Refresh() error {
	return nil
}

// HandleInput handles keyboard input
func (v *FaultMenuView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleListNavigation(v.list, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}

// FaultDetailsView displays detailed information about a fault
type FaultDetailsView struct {
	app       *App
	projectID int
	faultID   int
	textView  *tview.TextView
	fault     *hbapi.Fault
}

// NewFaultDetailsView creates a new fault details view
func NewFaultDetailsView(app *App, projectID, faultID int) *FaultDetailsView {
	v := &FaultDetailsView{
		app:       app,
		projectID: projectID,
		faultID:   faultID,
		textView:  tview.NewTextView(),
	}
	v.setupView()
	return v
}

func (v *FaultDetailsView) setupView() {
	v.textView.SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)

	v.textView.SetTitle(" Fault Details ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)
}

// Name returns the view name
func (v *FaultDetailsView) Name() string {
	return "Details"
}

// Render returns the view's primitive
func (v *FaultDetailsView) Render() tview.Primitive {
	return v.textView
}

// Refresh reloads the data
func (v *FaultDetailsView) Refresh() error {
	fault, err := v.app.Client().Faults.Get(v.app.Context(), v.projectID, v.faultID)
	if err != nil {
		return fmt.Errorf("failed to get fault: %w", err)
	}

	v.fault = fault
	v.app.Application().QueueUpdateDraw(func() {
		v.renderDetails()
	})
	return nil
}

func (v *FaultDetailsView) renderDetails() {
	if v.fault == nil {
		return
	}

	f := v.fault
	text := fmt.Sprintf(`[yellow]Class:[white] %s

[yellow]Message:[white] %s

[yellow]Environment:[white] %s
[yellow]Component:[white] %s
[yellow]Action:[white] %s

[yellow]Created:[white] %s
[yellow]Last Notice:[white] %s

[yellow]Notice Count:[white] %d
[yellow]Comments Count:[white] %d

[yellow]Status:[white] `,
		f.Klass,
		f.Message,
		f.Environment,
		f.Component,
		f.Action,
		f.CreatedAt.Format("2006-01-02 15:04:05"),
		formatTime(f.LastNoticeAt),
		f.NoticesCount,
		f.CommentsCount,
	)

	if f.Resolved {
		text += "[green]Resolved[white]"
	} else if f.Ignored {
		text += "[yellow]Ignored[white]"
	} else {
		text += "[red]Active[white]"
	}

	if f.Assignee != nil {
		text += fmt.Sprintf("\n\n[yellow]Assignee:[white] %s <%s>", f.Assignee.Name, f.Assignee.Email)
	}

	if len(f.Tags) > 0 {
		text += fmt.Sprintf("\n\n[yellow]Tags:[white] %v", f.Tags)
	}

	text += fmt.Sprintf("\n\n[yellow]URL:[white] %s", f.URL)

	v.textView.SetText(text)
}

// HandleInput handles keyboard input
func (v *FaultDetailsView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}

// NoticesView displays notices for a fault
type NoticesView struct {
	app       *App
	projectID int
	faultID   int
	table     *tview.Table
	notices   []hbapi.Notice
}

// NewNoticesView creates a new notices view
func NewNoticesView(app *App, projectID, faultID int) *NoticesView {
	v := &NoticesView{
		app:       app,
		projectID: projectID,
		faultID:   faultID,
		table:     tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *NoticesView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Notices ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"ID", "MESSAGE", "ENVIRONMENT", "HOSTNAME", "CREATED"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}
}

// Name returns the view name
func (v *NoticesView) Name() string {
	return "Notices"
}

// Render returns the view's primitive
func (v *NoticesView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *NoticesView) Refresh() error {
	response, err := v.app.Client().Faults.ListNotices(v.app.Context(), v.projectID, v.faultID, hbapi.FaultListNoticesOptions{
		Limit: 25,
	})
	if err != nil {
		return fmt.Errorf("failed to list notices: %w", err)
	}

	v.notices = response.Results
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *NoticesView) renderTable() {
	clearTableRows(v.table)

	for i, notice := range v.notices {
		row := i + 1

		message := truncateString(notice.Message, 50)
		id := truncateString(notice.ID, 15)

		v.table.SetCell(row, 0, tview.NewTableCell(id).SetExpansion(1))
		v.table.SetCell(row, 1, tview.NewTableCell(message).SetExpansion(3))
		v.table.SetCell(row, 2, tview.NewTableCell(notice.EnvironmentName).SetExpansion(1))
		v.table.SetCell(row, 3, tview.NewTableCell(notice.Environment.Hostname).SetExpansion(2))
		v.table.SetCell(row, 4, tview.NewTableCell(notice.CreatedAt.Format("2006-01-02 15:04")).SetExpansion(2))
	}

	if len(v.notices) > 0 {
		v.table.Select(1, 0)
	}
}

// HandleInput handles keyboard input
func (v *NoticesView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}

// AffectedUsersView displays affected users for a fault
type AffectedUsersView struct {
	app       *App
	projectID int
	faultID   int
	table     *tview.Table
	users []hbapi.FaultAffectedUser
}

// NewAffectedUsersView creates a new affected users view
func NewAffectedUsersView(app *App, projectID, faultID int) *AffectedUsersView {
	v := &AffectedUsersView{
		app:       app,
		projectID: projectID,
		faultID:   faultID,
		table:     tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *AffectedUsersView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Affected Users ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"USER", "OCCURRENCES"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}
}

// Name returns the view name
func (v *AffectedUsersView) Name() string {
	return "Affected Users"
}

// Render returns the view's primitive
func (v *AffectedUsersView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *AffectedUsersView) Refresh() error {
	users, err := v.app.Client().Faults.ListAffectedUsers(v.app.Context(), v.projectID, v.faultID, hbapi.FaultListAffectedUsersOptions{})
	if err != nil {
		return fmt.Errorf("failed to list affected users: %w", err)
	}

	v.users = users
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *AffectedUsersView) renderTable() {
	clearTableRows(v.table)

	for i, user := range v.users {
		row := i + 1
		v.table.SetCell(row, 0, tview.NewTableCell(user.User).SetExpansion(3))
		v.table.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%d", user.Count)).SetExpansion(1))
	}

	if len(v.users) > 0 {
		v.table.Select(1, 0)
	}
}

// HandleInput handles keyboard input
func (v *AffectedUsersView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}
