package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	hbapi "github.com/honeybadger-io/api-go"
	"github.com/rivo/tview"
)

// TeamsView displays a list of teams for an account
type TeamsView struct {
	app       *App
	accountID string
	table     *tview.Table
	teams     []hbapi.Team
}

// NewTeamsView creates a new teams view
func NewTeamsView(app *App, accountID string) *TeamsView {
	v := &TeamsView{
		app:       app,
		accountID: accountID,
		table:     tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *TeamsView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Teams ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"ID", "NAME", "CREATED"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}

	v.table.SetSelectedFunc(func(row, _ int) {
		if row > 0 && row <= len(v.teams) {
			team := v.teams[row-1]
			v.drillDown(team)
		}
	})
}

func (v *TeamsView) drillDown(team hbapi.Team) {
	menuView := NewTeamMenuView(v.app, team)
	v.app.Push(menuView)
}

// Name returns the view name
func (v *TeamsView) Name() string {
	return "Teams"
}

// Render returns the view's primitive
func (v *TeamsView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *TeamsView) Refresh() error {
	teams, err := v.app.Client().Teams.List(v.app.Context(), v.accountID)
	if err != nil {
		return fmt.Errorf("failed to list teams: %w", err)
	}

	v.teams = teams
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *TeamsView) renderTable() {
	clearTableRows(v.table)

	if len(v.teams) == 0 {
		showEmptyState(v.table, "No teams found")
		return
	}

	for i, team := range v.teams {
		row := i + 1
		v.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", team.ID)).SetExpansion(1))
		v.table.SetCell(row, 1, tview.NewTableCell(team.Name).SetExpansion(3))
		v.table.SetCell(
			row,
			2,
			tview.NewTableCell(team.CreatedAt.Format("2006-01-02 15:04")).SetExpansion(2),
		)
	}

	v.table.Select(1, 0)
}

// HandleInput handles keyboard input
func (v *TeamsView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	if isSelectKey(event) {
		row, _ := v.table.GetSelection()
		if row > 0 && row <= len(v.teams) {
			v.drillDown(v.teams[row-1])
		}
		return nil
	}
	return event
}

// TeamMenuView shows options for a selected team
type TeamMenuView struct {
	app  *App
	team hbapi.Team
	list *tview.List
}

// NewTeamMenuView creates a new team menu view
func NewTeamMenuView(app *App, team hbapi.Team) *TeamMenuView {
	v := &TeamMenuView{
		app:  app,
		team: team,
		list: tview.NewList(),
	}
	v.setupList()
	return v
}

func (v *TeamMenuView) setupList() {
	v.list.SetTitle(fmt.Sprintf(" %s ", v.team.Name)).
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	v.list.AddItem("Members", "View team members", 'm', func() {
		membersView := NewTeamMembersView(v.app, v.team.ID)
		v.app.Push(membersView)
	})

	v.list.AddItem("Invitations", "View team invitations", 'i', func() {
		invitationsView := NewTeamInvitationsView(v.app, v.team.ID)
		v.app.Push(invitationsView)
	})

	v.list.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
}

// Name returns the view name
func (v *TeamMenuView) Name() string {
	return v.team.Name
}

// Render returns the view's primitive
func (v *TeamMenuView) Render() tview.Primitive {
	return v.list
}

// Refresh reloads the data
func (v *TeamMenuView) Refresh() error {
	return nil
}

// HandleInput handles keyboard input
func (v *TeamMenuView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleListNavigation(v.list, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}

// TeamMembersView displays members for a team
type TeamMembersView struct {
	app     *App
	teamID  int
	table   *tview.Table
	members []hbapi.TeamMember
}

// NewTeamMembersView creates a new team members view
func NewTeamMembersView(app *App, teamID int) *TeamMembersView {
	v := &TeamMembersView{
		app:    app,
		teamID: teamID,
		table:  tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *TeamMembersView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Team Members ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"ID", "NAME", "EMAIL", "ADMIN"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}
}

// Name returns the view name
func (v *TeamMembersView) Name() string {
	return "Members"
}

// Render returns the view's primitive
func (v *TeamMembersView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *TeamMembersView) Refresh() error {
	members, err := v.app.Client().Teams.ListMembers(v.app.Context(), v.teamID)
	if err != nil {
		return fmt.Errorf("failed to list team members: %w", err)
	}

	v.members = members
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *TeamMembersView) renderTable() {
	clearTableRows(v.table)

	if len(v.members) == 0 {
		showEmptyState(v.table, "No team members found")
		return
	}

	for i, member := range v.members {
		row := i + 1

		admin := "No"
		if member.Admin {
			admin = "Yes"
		}

		v.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", member.ID)).SetExpansion(1))
		v.table.SetCell(row, 1, tview.NewTableCell(member.Name).SetExpansion(2))
		v.table.SetCell(row, 2, tview.NewTableCell(member.Email).SetExpansion(2))
		v.table.SetCell(row, 3, tview.NewTableCell(admin).SetExpansion(1))
	}

	v.table.Select(1, 0)
}

// HandleInput handles keyboard input
func (v *TeamMembersView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}

// TeamInvitationsView displays invitations for a team
type TeamInvitationsView struct {
	app         *App
	teamID      int
	table       *tview.Table
	invitations []hbapi.TeamInvitation
}

// NewTeamInvitationsView creates a new team invitations view
func NewTeamInvitationsView(app *App, teamID int) *TeamInvitationsView {
	v := &TeamInvitationsView{
		app:    app,
		teamID: teamID,
		table:  tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *TeamInvitationsView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Team Invitations ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"ID", "EMAIL", "ADMIN", "CREATED", "ACCEPTED"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}
}

// Name returns the view name
func (v *TeamInvitationsView) Name() string {
	return "Invitations"
}

// Render returns the view's primitive
func (v *TeamInvitationsView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *TeamInvitationsView) Refresh() error {
	invitations, err := v.app.Client().Teams.ListInvitations(v.app.Context(), v.teamID)
	if err != nil {
		return fmt.Errorf("failed to list team invitations: %w", err)
	}

	v.invitations = invitations
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *TeamInvitationsView) renderTable() {
	clearTableRows(v.table)

	if len(v.invitations) == 0 {
		showEmptyState(v.table, "No invitations found")
		return
	}

	for i, inv := range v.invitations {
		row := i + 1

		admin := "No"
		if inv.Admin {
			admin = "Yes"
		}

		accepted := "No"
		if inv.AcceptedAt != nil {
			accepted = inv.AcceptedAt.Format("2006-01-02")
		}

		v.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", inv.ID)).SetExpansion(1))
		v.table.SetCell(row, 1, tview.NewTableCell(inv.Email).SetExpansion(2))
		v.table.SetCell(row, 2, tview.NewTableCell(admin).SetExpansion(1))
		v.table.SetCell(
			row,
			3,
			tview.NewTableCell(inv.CreatedAt.Format("2006-01-02")).SetExpansion(1),
		)
		v.table.SetCell(row, 4, tview.NewTableCell(accepted).SetExpansion(1))
	}

	v.table.Select(1, 0)
}

// HandleInput handles keyboard input
func (v *TeamInvitationsView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}
