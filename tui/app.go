package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	hbapi "github.com/honeybadger-io/api-go"
	"github.com/rivo/tview"
)

// App represents the main TUI application
type App struct {
	app       *tview.Application
	client    *hbapi.Client
	pages     *tview.Pages
	header    *tview.TextView
	footer    *tview.TextView
	mainFlex  *tview.Flex
	navStack  []View
	ctx       context.Context
	cancel    context.CancelFunc
}

// View interface for all views in the TUI
type View interface {
	Name() string
	Render() tview.Primitive
	Refresh() error
	HandleInput(event *tcell.EventKey) *tcell.EventKey
}

// NewApp creates a new TUI application
func NewApp(client *hbapi.Client) *App {
	ctx, cancel := context.WithCancel(context.Background())
	a := &App{
		app:      tview.NewApplication(),
		client:   client,
		pages:    tview.NewPages(),
		navStack: make([]View, 0),
		ctx:      ctx,
		cancel:   cancel,
	}

	a.setupLayout()
	return a
}

func (a *App) setupLayout() {
	// Header showing breadcrumb navigation
	a.header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	a.header.SetBorder(false).
		SetBackgroundColor(tcell.ColorDarkBlue)

	// Footer showing help
	a.footer = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[yellow]↑↓/jk[white] Navigate  [yellow]Enter/→[white] Select  [yellow]Esc/←[white] Back  [yellow]r[white] Refresh  [yellow]q[white] Quit  [yellow]?[white] Help")
	a.footer.SetBorder(false).
		SetBackgroundColor(tcell.ColorDarkBlue)

	// Main layout
	a.mainFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.header, 1, 0, false).
		AddItem(a.pages, 0, 1, true).
		AddItem(a.footer, 1, 0, false)

	a.app.SetRoot(a.mainFlex, true)

	// Global input handling
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyLeft:
			if len(a.navStack) > 1 {
				a.Pop()
				return nil
			}
		case tcell.KeyCtrlC:
			a.app.Stop()
			return nil
		}

		switch event.Rune() {
		case 'q':
			if len(a.navStack) > 1 {
				a.Pop()
			} else {
				a.app.Stop()
			}
			return nil
		case 'r':
			if len(a.navStack) > 0 {
				currentView := a.navStack[len(a.navStack)-1]
				if err := currentView.Refresh(); err != nil {
					a.ShowError(err)
				}
			}
			return nil
		case '?':
			a.ShowHelp()
			return nil
		}

		// Pass to current view
		if len(a.navStack) > 0 {
			currentView := a.navStack[len(a.navStack)-1]
			return currentView.HandleInput(event)
		}

		return event
	})
}

// Push adds a new view to the navigation stack
func (a *App) Push(view View) {
	a.navStack = append(a.navStack, view)
	pageName := fmt.Sprintf("page-%d", len(a.navStack))
	a.pages.AddPage(pageName, view.Render(), true, true)
	a.pages.SwitchToPage(pageName)
	a.updateHeader()

	// Refresh the view to load data in a goroutine with cancellation support
	go func() {
		// Check if context is cancelled before starting
		select {
		case <-a.ctx.Done():
			return
		default:
		}

		if err := view.Refresh(); err != nil {
			// Check if context is cancelled before showing error
			select {
			case <-a.ctx.Done():
				return
			default:
				a.app.QueueUpdateDraw(func() {
					a.ShowError(err)
				})
			}
		}

		// Check if context is cancelled before drawing
		select {
		case <-a.ctx.Done():
			return
		default:
			a.app.Draw()
		}
	}()
}

// Pop removes the current view from the navigation stack
func (a *App) Pop() {
	if len(a.navStack) <= 1 {
		return
	}

	// Remove current page
	pageName := fmt.Sprintf("page-%d", len(a.navStack))
	a.pages.RemovePage(pageName)

	// Pop from stack
	a.navStack = a.navStack[:len(a.navStack)-1]

	// Switch to previous page
	prevPageName := fmt.Sprintf("page-%d", len(a.navStack))
	a.pages.SwitchToPage(prevPageName)
	a.updateHeader()
}

func (a *App) updateHeader() {
	var parts []string
	for _, v := range a.navStack {
		parts = append(parts, v.Name())
	}
	breadcrumb := strings.Join(parts, " > ")
	a.header.SetText(fmt.Sprintf(" [yellow]Honeybadger[white] │ %s", breadcrumb))
}

// ShowError displays an error modal
func (a *App) ShowError(err error) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Error: %v", err)).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("error")
		})
	a.pages.AddPage("error", modal, true, true)
}

// ShowHelp displays the help modal
func (a *App) ShowHelp() {
	helpText := `[yellow]Honeybadger TUI Help[white]

[green]Navigation:[white]
  ↑/k        Move up
  ↓/j        Move down
  Enter/→/l  Select/Drill down
  Esc/←/h    Go back
  q          Quit (or go back)

[green]Actions:[white]
  r          Refresh current view
  /          Search (in list views)
  ?          Show this help

[green]Views:[white]
  Accounts → Projects → Faults/Deployments/Uptime/etc.

Press any key to close this help.`

	modal := tview.NewModal().
		SetText(helpText).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("help")
		})
	a.pages.AddPage("help", modal, true, true)
}

// Run starts the TUI application
func (a *App) Run() error {
	// Ensure context is cancelled when the app stops
	defer a.cancel()

	// Start with accounts view
	accountsView := NewAccountsView(a)
	a.Push(accountsView)

	return a.app.Run()
}

// Client returns the API client
func (a *App) Client() *hbapi.Client {
	return a.client
}

// Context returns the context
func (a *App) Context() context.Context {
	return a.ctx
}

// Application returns the tview application
func (a *App) Application() *tview.Application {
	return a.app
}
