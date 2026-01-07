package tui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	hbapi "github.com/honeybadger-io/api-go"
	"github.com/rivo/tview"
)

// DeploymentsView displays deployments for a project
type DeploymentsView struct {
	app         *App
	projectID   int
	table       *tview.Table
	deployments []hbapi.Deployment
}

// NewDeploymentsView creates a new deployments view
func NewDeploymentsView(app *App, projectID int) *DeploymentsView {
	v := &DeploymentsView{
		app:       app,
		projectID: projectID,
		table:     tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *DeploymentsView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Deployments ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"ID", "ENVIRONMENT", "REVISION", "USER", "REPOSITORY", "CREATED"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}

	v.table.SetSelectedFunc(func(row, col int) {
		if row > 0 && row <= len(v.deployments) {
			deployment := v.deployments[row-1]
			v.showDetails(deployment)
		}
	})
}

func (v *DeploymentsView) showDetails(deployment hbapi.Deployment) {
	detailsView := NewDeploymentDetailsView(v.app, deployment)
	v.app.Push(detailsView)
}

// Name returns the view name
func (v *DeploymentsView) Name() string {
	return "Deployments"
}

// Render returns the view's primitive
func (v *DeploymentsView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *DeploymentsView) Refresh() error {
	deployments, err := v.app.Client().Deployments.List(v.app.Context(), v.projectID, hbapi.DeploymentListOptions{
		Limit: 25,
	})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	v.deployments = deployments
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *DeploymentsView) renderTable() {
	clearTableRows(v.table)

	for i, d := range v.deployments {
		row := i + 1

		revision := truncateString(d.Revision, 12)
		repo := truncateString(d.Repository, 30)

		v.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", d.ID)).SetExpansion(1))
		v.table.SetCell(row, 1, tview.NewTableCell(d.Environment).SetExpansion(1))
		v.table.SetCell(row, 2, tview.NewTableCell(revision).SetExpansion(1))
		v.table.SetCell(row, 3, tview.NewTableCell(d.LocalUsername).SetExpansion(1))
		v.table.SetCell(row, 4, tview.NewTableCell(repo).SetExpansion(2))
		v.table.SetCell(row, 5, tview.NewTableCell(d.CreatedAt.Format("2006-01-02 15:04")).SetExpansion(2))
	}

	if len(v.deployments) > 0 {
		v.table.Select(1, 0)
	}
}

// HandleInput handles keyboard input
func (v *DeploymentsView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	if isSelectKey(event) {
		row, _ := v.table.GetSelection()
		if row > 0 && row <= len(v.deployments) {
			v.showDetails(v.deployments[row-1])
		}
		return nil
	}
	return event
}

// DeploymentDetailsView displays detailed information about a deployment
type DeploymentDetailsView struct {
	app        *App
	deployment hbapi.Deployment
	textView   *tview.TextView
}

// NewDeploymentDetailsView creates a new deployment details view
func NewDeploymentDetailsView(app *App, deployment hbapi.Deployment) *DeploymentDetailsView {
	v := &DeploymentDetailsView{
		app:        app,
		deployment: deployment,
		textView:   tview.NewTextView(),
	}
	v.setupView()
	return v
}

func (v *DeploymentDetailsView) setupView() {
	v.textView.SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)

	v.textView.SetTitle(" Deployment Details ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)
}

// Name returns the view name
func (v *DeploymentDetailsView) Name() string {
	return "Details"
}

// Render returns the view's primitive
func (v *DeploymentDetailsView) Render() tview.Primitive {
	return v.textView
}

// Refresh reloads the data
func (v *DeploymentDetailsView) Refresh() error {
	v.app.Application().QueueUpdateDraw(func() {
		v.renderDetails()
	})
	return nil
}

func (v *DeploymentDetailsView) renderDetails() {
	d := v.deployment

	text := fmt.Sprintf(`[yellow]ID:[white] %d

[yellow]Environment:[white] %s

[yellow]Revision:[white] %s

[yellow]Repository:[white] %s

[yellow]Local Username:[white] %s

[yellow]Project ID:[white] %d

[yellow]Created:[white] %s`,
		d.ID,
		d.Environment,
		d.Revision,
		d.Repository,
		d.LocalUsername,
		d.ProjectID,
		d.CreatedAt.Format("2006-01-02 15:04:05"),
	)

	v.textView.SetText(text)
}

// HandleInput handles keyboard input
func (v *DeploymentDetailsView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}

// Helper function to format time pointers
func formatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}
