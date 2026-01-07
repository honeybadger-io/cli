package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	hbapi "github.com/honeybadger-io/api-go"
	"github.com/rivo/tview"
)

// UptimeSitesView displays uptime sites for a project
type UptimeSitesView struct {
	app       *App
	projectID int
	table     *tview.Table
	sites     []hbapi.Site
}

// NewUptimeSitesView creates a new uptime sites view
func NewUptimeSitesView(app *App, projectID int) *UptimeSitesView {
	v := &UptimeSitesView{
		app:       app,
		projectID: projectID,
		table:     tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *UptimeSitesView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Uptime Sites ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"ID", "NAME", "URL", "STATE", "ACTIVE", "FREQUENCY"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}

	v.table.SetSelectedFunc(func(row, _ int) {
		if row > 0 && row <= len(v.sites) {
			site := v.sites[row-1]
			v.drillDown(site)
		}
	})
}

func (v *UptimeSitesView) drillDown(site hbapi.Site) {
	menuView := NewSiteMenuView(v.app, v.projectID, site)
	v.app.Push(menuView)
}

// Name returns the view name
func (v *UptimeSitesView) Name() string {
	return "Uptime Sites"
}

// Render returns the view's primitive
func (v *UptimeSitesView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *UptimeSitesView) Refresh() error {
	sites, err := v.app.Client().Uptime.List(v.app.Context(), v.projectID)
	if err != nil {
		return fmt.Errorf("failed to list uptime sites: %w", err)
	}

	v.sites = sites
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *UptimeSitesView) renderTable() {
	clearTableRows(v.table)

	if len(v.sites) == 0 {
		showEmptyState(v.table, "No uptime sites found")
		return
	}

	for i, site := range v.sites {
		row := i + 1

		active := "No"
		if site.Active {
			active = "Yes"
		}

		stateColor := tcell.ColorGreen
		if site.State == "down" {
			stateColor = tcell.ColorRed
		}

		url := truncateString(site.URL, 40)

		v.table.SetCell(row, 0, tview.NewTableCell(site.ID).SetExpansion(1))
		v.table.SetCell(row, 1, tview.NewTableCell(site.Name).SetExpansion(2))
		v.table.SetCell(row, 2, tview.NewTableCell(url).SetExpansion(3))
		v.table.SetCell(
			row,
			3,
			tview.NewTableCell(site.State).SetTextColor(stateColor).SetExpansion(1),
		)
		v.table.SetCell(row, 4, tview.NewTableCell(active).SetExpansion(1))
		v.table.SetCell(
			row,
			5,
			tview.NewTableCell(fmt.Sprintf("%dm", site.Frequency)).SetExpansion(1),
		)
	}

	v.table.Select(1, 0)
	v.table.ScrollToBeginning()
}

// HandleInput handles keyboard input
func (v *UptimeSitesView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	if isSelectKey(event) {
		row, _ := v.table.GetSelection()
		if row > 0 && row <= len(v.sites) {
			v.drillDown(v.sites[row-1])
		}
		return nil
	}
	return event
}

// SiteMenuView shows options for a selected site
type SiteMenuView struct {
	app       *App
	projectID int
	site      hbapi.Site
	list      *tview.List
}

// NewSiteMenuView creates a new site menu view
func NewSiteMenuView(app *App, projectID int, site hbapi.Site) *SiteMenuView {
	v := &SiteMenuView{
		app:       app,
		projectID: projectID,
		site:      site,
		list:      tview.NewList(),
	}
	v.setupList()
	return v
}

func (v *SiteMenuView) setupList() {
	v.list.SetTitle(fmt.Sprintf(" %s ", v.site.Name)).
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	v.list.AddItem("Details", "View site details", 'd', func() {
		detailsView := NewSiteDetailsView(v.app, v.projectID, v.site.ID)
		v.app.Push(detailsView)
	})

	v.list.AddItem("Outages", "View outage history", 'o', func() {
		outagesView := NewOutagesView(v.app, v.projectID, v.site.ID)
		v.app.Push(outagesView)
	})

	v.list.AddItem("Checks", "View uptime checks", 'c', func() {
		checksView := NewUptimeChecksView(v.app, v.projectID, v.site.ID)
		v.app.Push(checksView)
	})

	v.list.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
}

// Name returns the view name
func (v *SiteMenuView) Name() string {
	return v.site.Name
}

// Render returns the view's primitive
func (v *SiteMenuView) Render() tview.Primitive {
	return v.list
}

// Refresh reloads the data
func (v *SiteMenuView) Refresh() error {
	return nil
}

// HandleInput handles keyboard input
func (v *SiteMenuView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleListNavigation(v.list, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}

// SiteDetailsView displays detailed information about a site
type SiteDetailsView struct {
	app       *App
	projectID int
	siteID    string
	textView  *tview.TextView
	site      *hbapi.Site
}

// NewSiteDetailsView creates a new site details view
func NewSiteDetailsView(app *App, projectID int, siteID string) *SiteDetailsView {
	v := &SiteDetailsView{
		app:       app,
		projectID: projectID,
		siteID:    siteID,
		textView:  tview.NewTextView(),
	}
	v.setupView()
	return v
}

func (v *SiteDetailsView) setupView() {
	v.textView.SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)

	v.textView.SetTitle(" Site Details ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)
}

// Name returns the view name
func (v *SiteDetailsView) Name() string {
	return "Details"
}

// Render returns the view's primitive
func (v *SiteDetailsView) Render() tview.Primitive {
	return v.textView
}

// Refresh reloads the data
func (v *SiteDetailsView) Refresh() error {
	site, err := v.app.Client().Uptime.Get(v.app.Context(), v.projectID, v.siteID)
	if err != nil {
		return fmt.Errorf("failed to get site: %w", err)
	}

	v.site = site
	v.app.Application().QueueUpdateDraw(func() {
		v.renderDetails()
	})
	return nil
}

func (v *SiteDetailsView) renderDetails() {
	if v.site == nil {
		return
	}

	s := v.site
	stateColor := "green"
	if s.State == "down" {
		stateColor = "red"
	}

	text := fmt.Sprintf(`[yellow]Name:[white] %s

[yellow]URL:[white] %s

[yellow]State:[%s] %s[white]
[yellow]Active:[white] %v
[yellow]Frequency:[white] %d minutes

[yellow]Match Type:[white] %s`,
		s.Name,
		s.URL,
		stateColor, s.State,
		s.Active,
		s.Frequency,
		s.MatchType,
	)

	if s.Match != nil {
		text += fmt.Sprintf("\n[yellow]Match:[white] %s", *s.Match)
	}

	if s.LastCheckedAt != nil {
		text += fmt.Sprintf(
			"\n\n[yellow]Last Checked:[white] %s",
			s.LastCheckedAt.Format("2006-01-02 15:04:05"),
		)
	}

	v.textView.SetText(text)
}

// HandleInput handles keyboard input
func (v *SiteDetailsView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}

// OutagesView displays outages for a site
type OutagesView struct {
	app       *App
	projectID int
	siteID    string
	table     *tview.Table
	outages   []hbapi.Outage
}

// NewOutagesView creates a new outages view
func NewOutagesView(app *App, projectID int, siteID string) *OutagesView {
	v := &OutagesView{
		app:       app,
		projectID: projectID,
		siteID:    siteID,
		table:     tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *OutagesView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Outages ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"DOWN AT", "UP AT", "STATUS", "REASON"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}
}

// Name returns the view name
func (v *OutagesView) Name() string {
	return "Outages"
}

// Render returns the view's primitive
func (v *OutagesView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *OutagesView) Refresh() error {
	outages, err := v.app.Client().Uptime.ListOutages(
		v.app.Context(),
		v.projectID,
		v.siteID,
		hbapi.OutageListOptions{
			Limit: 25,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to list outages: %w", err)
	}

	v.outages = outages
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *OutagesView) renderTable() {
	clearTableRows(v.table)

	if len(v.outages) == 0 {
		showEmptyState(v.table, "No outages found")
		return
	}

	for i, outage := range v.outages {
		row := i + 1

		upAt := "Still down"
		if outage.UpAt != nil {
			upAt = outage.UpAt.Format("2006-01-02 15:04")
		}

		reason := truncateString(outage.Reason, 40)

		v.table.SetCell(
			row,
			0,
			tview.NewTableCell(outage.DownAt.Format("2006-01-02 15:04")).SetExpansion(2),
		)
		v.table.SetCell(row, 1, tview.NewTableCell(upAt).SetExpansion(2))
		v.table.SetCell(
			row,
			2,
			tview.NewTableCell(fmt.Sprintf("%d", outage.Status)).SetExpansion(1),
		)
		v.table.SetCell(row, 3, tview.NewTableCell(reason).SetExpansion(3))
	}

	v.table.Select(1, 0)
	v.table.ScrollToBeginning()
}

// HandleInput handles keyboard input
func (v *OutagesView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}

// UptimeChecksView displays uptime checks for a site
type UptimeChecksView struct {
	app       *App
	projectID int
	siteID    string
	table     *tview.Table
	checks    []hbapi.UptimeCheck
}

// NewUptimeChecksView creates a new uptime checks view
func NewUptimeChecksView(app *App, projectID int, siteID string) *UptimeChecksView {
	v := &UptimeChecksView{
		app:       app,
		projectID: projectID,
		siteID:    siteID,
		table:     tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *UptimeChecksView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Uptime Checks ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"CREATED", "LOCATION", "UP", "DURATION"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}
}

// Name returns the view name
func (v *UptimeChecksView) Name() string {
	return "Checks"
}

// Render returns the view's primitive
func (v *UptimeChecksView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *UptimeChecksView) Refresh() error {
	checks, err := v.app.Client().Uptime.ListUptimeChecks(
		v.app.Context(),
		v.projectID,
		v.siteID,
		hbapi.UptimeCheckListOptions{
			Limit: 25,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to list uptime checks: %w", err)
	}

	v.checks = checks
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *UptimeChecksView) renderTable() {
	clearTableRows(v.table)

	if len(v.checks) == 0 {
		showEmptyState(v.table, "No uptime checks found")
		return
	}

	for i, check := range v.checks {
		row := i + 1

		up := "No"
		upColor := tcell.ColorRed
		if check.Up {
			up = "Yes"
			upColor = tcell.ColorGreen
		}

		v.table.SetCell(
			row,
			0,
			tview.NewTableCell(check.CreatedAt.Format("2006-01-02 15:04:05")).SetExpansion(2),
		)
		v.table.SetCell(row, 1, tview.NewTableCell(check.Location).SetExpansion(2))
		v.table.SetCell(row, 2, tview.NewTableCell(up).SetTextColor(upColor).SetExpansion(1))
		v.table.SetCell(
			row,
			3,
			tview.NewTableCell(fmt.Sprintf("%dms", check.Duration)).SetExpansion(1),
		)
	}

	v.table.Select(1, 0)
	v.table.ScrollToBeginning()
}

// HandleInput handles keyboard input
func (v *UptimeChecksView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}
