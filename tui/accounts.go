// Package tui provides a terminal user interface for browsing Honeybadger data.
package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	hbapi "github.com/honeybadger-io/api-go"
	"github.com/rivo/tview"
)

// AccountsView displays a list of accounts
type AccountsView struct {
	app      *App
	table    *tview.Table
	accounts []hbapi.Account
}

// NewAccountsView creates a new accounts view
func NewAccountsView(app *App) *AccountsView {
	v := &AccountsView{
		app:   app,
		table: tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *AccountsView) setupTable() {
	v.table.SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkCyan).
			Foreground(tcell.ColorWhite))

	v.table.SetTitle(" Accounts ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	// Header row
	headers := []string{"ID", "NAME", "EMAIL", "ACTIVE"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}

	// Handle selection
	v.table.SetSelectedFunc(func(row, _ int) {
		if row > 0 && row <= len(v.accounts) {
			account := v.accounts[row-1]
			v.drillDown(account)
		}
	})
}

func (v *AccountsView) drillDown(account hbapi.Account) {
	// Show account menu to choose what to view
	menuView := NewAccountMenuView(v.app, account)
	v.app.Push(menuView)
}

// Name returns the view name
func (v *AccountsView) Name() string {
	return "Accounts"
}

// Render returns the view's primitive
func (v *AccountsView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *AccountsView) Refresh() error {
	accounts, err := v.app.Client().Accounts.List(v.app.Context())
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}

	v.accounts = accounts
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *AccountsView) renderTable() {
	clearTableRows(v.table)

	if len(v.accounts) == 0 {
		showEmptyState(v.table, "No accounts found")
		return
	}

	// Add data rows
	for i, account := range v.accounts {
		row := i + 1

		active := "No"
		if account.Active != nil && *account.Active {
			active = "Yes"
		}

		v.table.SetCell(row, 0, tview.NewTableCell(account.ID).SetExpansion(1))
		v.table.SetCell(row, 1, tview.NewTableCell(account.Name).SetExpansion(2))
		v.table.SetCell(row, 2, tview.NewTableCell(account.Email).SetExpansion(2))
		v.table.SetCell(row, 3, tview.NewTableCell(active).SetExpansion(1))
	}

	v.table.Select(1, 0)
	v.table.ScrollToBeginning()
}

// HandleInput handles keyboard input
func (v *AccountsView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}

	if isSelectKey(event) {
		row, _ := v.table.GetSelection()
		if row > 0 && row <= len(v.accounts) {
			v.drillDown(v.accounts[row-1])
		}
		return nil
	}

	return event
}

// AccountMenuView shows options for a selected account
type AccountMenuView struct {
	app     *App
	account hbapi.Account
	list    *tview.List
}

// NewAccountMenuView creates a new account menu view
func NewAccountMenuView(app *App, account hbapi.Account) *AccountMenuView {
	v := &AccountMenuView{
		app:     app,
		account: account,
		list:    tview.NewList(),
	}
	v.setupList()
	return v
}

func (v *AccountMenuView) setupList() {
	v.list.SetTitle(fmt.Sprintf(" %s ", v.account.Name)).
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	v.list.AddItem("Projects", "View projects for this account", 'p', func() {
		projectsView := NewProjectsView(v.app, v.account.ID)
		v.app.Push(projectsView)
	})

	v.list.AddItem("Teams", "View teams for this account", 't', func() {
		teamsView := NewTeamsView(v.app, v.account.ID)
		v.app.Push(teamsView)
	})

	v.list.AddItem("Users", "View users for this account", 'u', func() {
		usersView := NewAccountUsersView(v.app, v.account.ID)
		v.app.Push(usersView)
	})

	v.list.AddItem("Invitations", "View pending invitations", 'i', func() {
		invitationsView := NewAccountInvitationsView(v.app, v.account.ID)
		v.app.Push(invitationsView)
	})

	v.list.AddItem("Status Pages", "View status pages for this account", 's', func() {
		statuspagesView := NewStatuspagesView(v.app, v.account.ID)
		v.app.Push(statuspagesView)
	})

	v.list.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
}

// Name returns the view name
func (v *AccountMenuView) Name() string {
	return v.account.Name
}

// Render returns the view's primitive
func (v *AccountMenuView) Render() tview.Primitive {
	return v.list
}

// Refresh reloads the data
func (v *AccountMenuView) Refresh() error {
	return nil
}

// HandleInput handles keyboard input
func (v *AccountMenuView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleListNavigation(v.list, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}

// AccountUsersView displays users for an account
type AccountUsersView struct {
	app       *App
	accountID string
	table     *tview.Table
	users     []hbapi.AccountUser
}

// NewAccountUsersView creates a new account users view
func NewAccountUsersView(app *App, accountID string) *AccountUsersView {
	v := &AccountUsersView{
		app:       app,
		accountID: accountID,
		table:     tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *AccountUsersView) setupTable() {
	setupReadOnlyTable(v.table)

	v.table.SetTitle(" Users ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"ID", "NAME", "EMAIL", "ROLE"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}
}

// Name returns the view name
func (v *AccountUsersView) Name() string {
	return "Users"
}

// Render returns the view's primitive
func (v *AccountUsersView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *AccountUsersView) Refresh() error {
	users, err := v.app.Client().Accounts.ListUsers(v.app.Context(), v.accountID)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	v.users = users
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *AccountUsersView) renderTable() {
	clearTableRows(v.table)

	if len(v.users) == 0 {
		showEmptyState(v.table, "No users found")
		return
	}

	for i, user := range v.users {
		row := i + 1
		v.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", user.ID)).SetExpansion(1))
		v.table.SetCell(row, 1, tview.NewTableCell(user.Name).SetExpansion(2))
		v.table.SetCell(row, 2, tview.NewTableCell(user.Email).SetExpansion(2))
		v.table.SetCell(row, 3, tview.NewTableCell(user.Role).SetExpansion(1))
	}

	v.table.Select(1, 0)
	v.table.ScrollToBeginning()
}

// HandleInput handles keyboard input
func (v *AccountUsersView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}

// AccountInvitationsView displays invitations for an account
type AccountInvitationsView struct {
	app         *App
	accountID   string
	table       *tview.Table
	invitations []hbapi.AccountInvitation
}

// NewAccountInvitationsView creates a new account invitations view
func NewAccountInvitationsView(app *App, accountID string) *AccountInvitationsView {
	v := &AccountInvitationsView{
		app:       app,
		accountID: accountID,
		table:     tview.NewTable(),
	}
	v.setupTable()
	return v
}

func (v *AccountInvitationsView) setupTable() {
	setupReadOnlyTable(v.table)

	v.table.SetTitle(" Invitations ").
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkCyan)

	headers := []string{"ID", "EMAIL", "ROLE", "CREATED", "ACCEPTED"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, col, cell)
	}
}

// Name returns the view name
func (v *AccountInvitationsView) Name() string {
	return "Invitations"
}

// Render returns the view's primitive
func (v *AccountInvitationsView) Render() tview.Primitive {
	return v.table
}

// Refresh reloads the data
func (v *AccountInvitationsView) Refresh() error {
	invitations, err := v.app.Client().Accounts.ListInvitations(v.app.Context(), v.accountID)
	if err != nil {
		return fmt.Errorf("failed to list invitations: %w", err)
	}

	v.invitations = invitations
	v.app.Application().QueueUpdateDraw(func() {
		v.renderTable()
	})
	return nil
}

func (v *AccountInvitationsView) renderTable() {
	clearTableRows(v.table)

	if len(v.invitations) == 0 {
		showEmptyState(v.table, "No invitations found")
		return
	}

	for i, inv := range v.invitations {
		row := i + 1
		accepted := "No"
		if inv.AcceptedAt != nil {
			accepted = inv.AcceptedAt.Format("2006-01-02")
		}

		v.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", inv.ID)).SetExpansion(1))
		v.table.SetCell(row, 1, tview.NewTableCell(inv.Email).SetExpansion(2))
		v.table.SetCell(row, 2, tview.NewTableCell(inv.Role).SetExpansion(1))
		v.table.SetCell(
			row,
			3,
			tview.NewTableCell(inv.CreatedAt.Format("2006-01-02")).SetExpansion(1),
		)
		v.table.SetCell(row, 4, tview.NewTableCell(accepted).SetExpansion(1))
	}

	v.table.Select(1, 0)
	v.table.ScrollToBeginning()
}

// HandleInput handles keyboard input
func (v *AccountInvitationsView) HandleInput(event *tcell.EventKey) *tcell.EventKey {
	if handleTableNavigation(v.table, event) {
		return nil
	}
	if handleBackNavigation(v.app, event) {
		return nil
	}
	return event
}
