package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	hbapi "github.com/honeybadger-io/api-go"
	"github.com/rivo/tview"
)

// ProjectsView displays a list of projects for an account
type ProjectsView struct {
	app       *App
	accountID string
	table     *tview.Table
	projects  []hbapi.Project
}

// NewProjectsView creates a new projects view
func NewProjectsView(app *App, accountID string) *ProjectsView {
	v := &ProjectsView{
		app:       app,
		accountID: accountID,
		table:     tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *ProjectsView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Projects ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"ID", "NAME", "ACTIVE", "FAULTS", "UNRESOLVED"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}

	v.table.SetSelectedFunc(func(row, _ int) {
		if row > 0 && row <= len(v.projects) {
			v.drillDown(v.projects[row-1])
		}
	})
}

func (v *ProjectsView) drillDown(project hbapi.Project) {
	menuView := NewProjectMenuView(v.app, project)
	v.app.Push(menuView)
}

// Name returns the view name
func (v *ProjectsView) Name() string {
	return "Projects"
}

// Render returns the view's primitive
func (v *ProjectsView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *ProjectsView) Refresh() error {
	response, err := v.app.Client().Projects.ListByAccountID(v.app.Context(), v.accountID)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	v.projects = response.Results
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *ProjectsView) renderTable() {
	clearTableRows(v.table)

	if len(v.projects) == 0 {
		showEmptyState(v.table, "No projects found")
		return
	}

	for i, project := range v.projects {
		row := i + 1

		active := "No"
		if project.Active {
			active = "Yes"
		}

		// Color unresolved faults red if > 0
		unresolvedCell := tview.NewTableCell(fmt.Sprintf("%d", project.UnresolvedFaultCount)).
			SetExpansion(1)
		if project.UnresolvedFaultCount > 0 {
			unresolvedCell.SetTextColor(tcell.ColorRed)
		}

		v.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", project.ID)).SetExpansion(1))
		v.table.SetCell(row, 1, tview.NewTableCell(project.Name).SetExpansion(2))
		v.table.SetCell(row, 2, tview.NewTableCell(active).SetExpansion(1))
		v.table.SetCell(
			row,
			3,
			tview.NewTableCell(fmt.Sprintf("%d", project.FaultCount)).SetExpansion(1),
		)
		v.table.SetCell(row, 4, unresolvedCell)
	}

	v.table.Select(1, 0)
	v.table.ScrollToBeginning()
}

// HandleInput handles keyboard input
func (v *ProjectsView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	if isSelectKey(event) {
		row, _ := v.table.GetSelection()
		if row > 0 && row <= len(v.projects) {
			v.drillDown(v.projects[row-1])
		}
		return nil
	}
	return event
}

// ProjectMenuView shows options for a selected project
type ProjectMenuView struct {
	app     *App
	project hbapi.Project
	list    *tview.List
}

// NewProjectMenuView creates a new project menu view
func NewProjectMenuView(app *App, project hbapi.Project) *ProjectMenuView {
	v := &ProjectMenuView{
		app:     app,
		project: project,
		list:    tview.NewList(),
	}
	v.setupList()
	return v
}

func (v *ProjectMenuView) setupList() {
	v.list.SetTitle(fmt.Sprintf(" %s ", v.project.Name)).
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	v.list.AddItem(
		"Faults",
		fmt.Sprintf("View faults (%d unresolved)", v.project.UnresolvedFaultCount),
		'f',
		func() {
			faultsView := NewFaultsView(v.app, v.project.ID)
			v.app.Push(faultsView)
		},
	)

	v.list.AddItem("Deployments", "View recent deployments", 'd', func() {
		deploymentsView := NewDeploymentsView(v.app, v.project.ID)
		v.app.Push(deploymentsView)
	})

	v.list.AddItem("Uptime Sites", "View uptime monitoring sites", 'u', func() {
		uptimeView := NewUptimeSitesView(v.app, v.project.ID)
		v.app.Push(uptimeView)
	})

	v.list.AddItem("Check-ins", "View check-ins (cron monitoring)", 'c', func() {
		checkinsView := NewCheckinsView(v.app, v.project.ID)
		v.app.Push(checkinsView)
	})

	v.list.AddItem("Integrations", "View notification integrations", 'i', func() {
		integrationsView := NewIntegrationsView(v.app, v.project.ID)
		v.app.Push(integrationsView)
	})

	v.list.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
}

// Name returns the view name
func (v *ProjectMenuView) Name() string {
	return v.project.Name
}

// Render returns the view's primitive
func (v *ProjectMenuView) Render() tview.Primitive {
	return v.list
}

// Refresh reloads the data
func (v *ProjectMenuView) Refresh() error {
	return nil
}

// HandleInput handles keyboard input
func (v *ProjectMenuView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleListNavigation(v.list, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}

// IntegrationsView displays integrations for a project
type IntegrationsView struct {
	app          *App
	projectID    int
	table        *tview.Table
	integrations []hbapi.ProjectIntegration
}

// NewIntegrationsView creates a new integrations view
func NewIntegrationsView(app *App, projectID int) *IntegrationsView {
	v := &IntegrationsView{
		app:       app,
		projectID: projectID,
		table:     tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *IntegrationsView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Integrations ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"ID", "TYPE", "ACTIVE", "EVENTS"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}
}

// Name returns the view name
func (v *IntegrationsView) Name() string {
	return "Integrations"
}

// Render returns the view's primitive
func (v *IntegrationsView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *IntegrationsView) Refresh() error {
	integrations, err := v.app.Client().Projects.GetIntegrations(v.app.Context(), v.projectID)
	if err != nil {
		return fmt.Errorf("failed to list integrations: %w", err)
	}

	v.integrations = integrations
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *IntegrationsView) renderTable() {
	clearTableRows(v.table)

	if len(v.integrations) == 0 {
		showEmptyState(v.table, "No integrations found")
		return
	}

	for i, integration := range v.integrations {
		row := i + 1

		active := "No"
		if integration.Active {
			active = "Yes"
		}

		events := "-"
		if len(integration.Events) > 0 {
			events = fmt.Sprintf("%v", integration.Events)
		}

		v.table.SetCell(
			row,
			0,
			tview.NewTableCell(fmt.Sprintf("%d", integration.ID)).SetExpansion(1),
		)
		v.table.SetCell(row, 1, tview.NewTableCell(integration.Type).SetExpansion(2))
		v.table.SetCell(row, 2, tview.NewTableCell(active).SetExpansion(1))
		v.table.SetCell(row, 3, tview.NewTableCell(events).SetExpansion(3))
	}

	v.table.Select(1, 0)
	v.table.ScrollToBeginning()
}

// HandleInput handles keyboard input
func (v *IntegrationsView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}
