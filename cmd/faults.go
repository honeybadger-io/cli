package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/oapi-codegen/nullable"
	"github.com/spf13/cobra"
)

var (
	faultsProjectID        string
	faultID                string
	faultQuery             string
	faultOrder             string
	faultLimit             int
	faultOutputFormat      string
	faultAffectedUserQuery string
	faultResolved          bool
	faultIgnored           bool
	faultResolveOnDeploy   bool
	faultAssigneeID        string
	faultUnassign          bool
	faultIDs               []string
	faultSelectAll         bool
	faultCreatedAfter      string
	faultOccurredAfter     string
	faultOccurredBefore    string
	faultDryRun            bool
)

// faultSummary is v3's fault summary, which apiv3 returns untyped.
type faultSummary struct {
	Total        int `json:"total"`
	Environments []struct {
		Environment *string `json:"environment"`
		Resolved    bool    `json:"resolved"`
		Ignored     bool    `json:"ignored"`
		Count       int     `json:"count"`
	} `json:"environments"`
}

// deref returns what p points to, or T's zero value when p is nil.
func deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// faultsCmd represents the faults command
var faultsCmd = &cobra.Command{
	Use:     "faults",
	Short:   "Manage Honeybadger faults",
	GroupID: GroupDataAPI,
	Long:    `View and manage faults (errors) in your Honeybadger projects.`,
}

// faultsListCmd represents the faults list command
var faultsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List faults for a project",
	Long:  `List all faults for a specific project with optional filtering.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&faultsProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		response, err := client.Faults.List(context.Background(), faultsProjectID,
			apiv3.Search(faultQuery),
			apiv3.OrderBy(faultOrder),
			apiv3.Limit(faultLimit))
		if err != nil {
			return dataAPIError("list faults", err)
		}

		if faultOutputFormat == "json" {
			return printJSON(response.Data)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tCLASS\tMESSAGE\tENV\tNOTICES\tRESOLVED\tLAST SEEN")
		for _, fault := range response.Data {
			lastSeen := "Never"
			if at, err := fault.LastNoticeAt.Get(); err == nil {
				lastSeen = at.Format("2006-01-02 15:04")
			}

			message := fault.Message
			if len(message) > 50 {
				message = message[:47] + "..."
			}

			resolved := " "
			if deref(fault.Resolved) {
				resolved = "✓"
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
				fault.Id,
				fault.Klass.GetOrEmpty(),
				message,
				fault.Environment.GetOrEmpty(),
				deref(fault.NoticesCount),
				resolved,
				lastSeen)
		}
		return w.Flush()
	},
}

// faultsGetCmd represents the faults get command
var faultsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a fault by ID",
	Long:  `Get detailed information about a specific fault.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&faultsProjectID); err != nil {
			return err
		}
		if faultID == "" {
			return fmt.Errorf("fault ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		fault, err := client.Faults.Get(context.Background(), faultsProjectID, faultID)
		if err != nil {
			return dataAPIError("get fault", err)
		}

		if faultOutputFormat == "json" {
			return printJSON(fault)
		}

		fmt.Printf("Fault Details:\n")
		fmt.Printf("  ID: %s\n", fault.Id)
		fmt.Printf("  Class: %s\n", fault.Klass.GetOrEmpty())
		fmt.Printf("  Message: %s\n", fault.Message)
		fmt.Printf("  Environment: %s\n", fault.Environment.GetOrEmpty())
		fmt.Printf("  Component: %s\n", fault.Component.GetOrEmpty())
		fmt.Printf("  Action: %s\n", fault.Action.GetOrEmpty())
		fmt.Printf("  Created: %s\n", fault.CreatedAt.Format("2006-01-02 15:04:05"))
		if at, err := fault.LastNoticeAt.Get(); err == nil {
			fmt.Printf("  Last Noticed: %s\n", at.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("  Notice Count: %d\n", deref(fault.NoticesCount))
		fmt.Printf("  Comments Count: %d\n", deref(fault.CommentsCount))
		fmt.Printf("  Resolved: %v\n", deref(fault.Resolved))
		fmt.Printf("  Ignored: %v\n", deref(fault.Ignored))
		fmt.Printf("  URL: %s\n", fault.Links.Web)
		if assignee, err := fault.Assignee.Get(); err == nil {
			fmt.Printf("  Assignee: %s <%s>\n", assignee.Name.GetOrEmpty(), deref(assignee.Email))
		}
		if tags := deref(fault.Tags); len(tags) > 0 {
			fmt.Printf("  Tags: %s\n", strings.Join(tags, ", "))
		}
		return nil
	},
}

// faultsNoticesCmd represents the faults notices command
var faultsNoticesCmd = &cobra.Command{
	Use:   "notices",
	Short: "List notices for a fault",
	Long:  `List individual error occurrences (notices) for a specific fault, newest first.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&faultsProjectID); err != nil {
			return err
		}
		if faultID == "" {
			return fmt.Errorf("fault ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		response, err := client.Faults.ListNotices(context.Background(), faultsProjectID, faultID,
			apiv3.Limit(faultLimit))
		if err != nil {
			return dataAPIError("list notices", err)
		}

		if faultOutputFormat == "json" {
			return printJSON(response.Data)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tMESSAGE\tENVIRONMENT\tCREATED")
		for _, notice := range response.Data {
			message := notice.Message.GetOrEmpty()
			if len(message) > 60 {
				message = message[:57] + "..."
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				notice.Id.String(),
				message,
				notice.Environment.GetOrEmpty(),
				notice.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		return w.Flush()
	},
}

// faultsUpdateCmd represents the faults update command
var faultsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a fault's state",
	Long: `Update a fault's resolved, ignored, assignee, or resolve-on-deploy state.

Only the flags you pass are updated; other fields are left unchanged. Note that
when --resolved or --ignored is set to true in the same request as
--resolve-on-deploy, the API gives the former precedence.

Examples:
  # Resolve a fault
  hb faults update --project-id Xk9mZp --id Fk7pQ2 --resolved

  # Un-resolve a fault
  hb faults update --project-id Xk9mZp --id Fk7pQ2 --resolved=false

  # Ignore a fault
  hb faults update --project-id Xk9mZp --id Fk7pQ2 --ignored

  # Mark a fault to be resolved automatically on the next deploy
  hb faults update --project-id Xk9mZp --id Fk7pQ2 --resolve-on-deploy

  # Assign a fault to a user
  hb faults update --project-id Xk9mZp --id Fk7pQ2 --assignee-id Us3rAb

  # Unassign a fault
  hb faults update --project-id Xk9mZp --id Fk7pQ2 --unassign`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&faultsProjectID); err != nil {
			return err
		}
		if faultID == "" {
			return fmt.Errorf("fault ID is required. Set it using --id flag")
		}
		if faultAssigneeID != "" && faultUnassign {
			return fmt.Errorf("--assignee-id and --unassign are mutually exclusive")
		}

		params := apiv3.FaultParams{}
		if cmd.Flags().Changed("resolved") {
			params.Resolved = &faultResolved
		}
		if cmd.Flags().Changed("ignored") {
			params.Ignored = &faultIgnored
		}
		if cmd.Flags().Changed("resolve-on-deploy") {
			params.ResolveOnDeploy = &faultResolveOnDeploy
		}
		if faultAssigneeID != "" {
			params.AssigneeId = nullable.NewNullableWithValue(faultAssigneeID)
		} else if faultUnassign {
			params.AssigneeId = nullable.NewNullNullable[string]()
		}

		if params.Resolved == nil && params.Ignored == nil && params.ResolveOnDeploy == nil &&
			!params.AssigneeId.IsSpecified() {
			return fmt.Errorf(
				"nothing to update. Set at least one of --resolved, --ignored, --resolve-on-deploy, --assignee-id, or --unassign",
			)
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		fault, err := client.Faults.Update(context.Background(), faultsProjectID, faultID, params)
		if err != nil {
			return dataAPIError("update fault", err)
		}

		fmt.Printf("Fault %s updated\n", fault.Id)
		return nil
	},
}

// faultsCountsCmd represents the faults counts command
var faultsCountsCmd = &cobra.Command{
	Use:   "counts",
	Short: "Get fault counts for a project",
	Long:  `Get summary counts of faults grouped by environment and status.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&faultsProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		summary, err := client.Faults.Summary(context.Background(), faultsProjectID)
		if err != nil {
			return dataAPIError("get fault counts", err)
		}

		if faultOutputFormat == "json" {
			return printJSON(summary)
		}

		raw, err := json.Marshal(summary)
		if err != nil {
			return fmt.Errorf("failed to read fault counts: %w", err)
		}
		var counts faultSummary
		if err := json.Unmarshal(raw, &counts); err != nil {
			return fmt.Errorf("failed to read fault counts: %w", err)
		}

		fmt.Printf("Total Faults: %d\n\n", counts.Total)
		if len(counts.Environments) == 0 {
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ENVIRONMENT\tRESOLVED\tIGNORED\tCOUNT")
		for _, env := range counts.Environments {
			_, _ = fmt.Fprintf(w, "%s\t%v\t%v\t%d\n",
				deref(env.Environment),
				env.Resolved,
				env.Ignored,
				env.Count)
		}
		return w.Flush()
	},
}

// faultsAffectedUsersCmd represents the faults affected-users command
var faultsAffectedUsersCmd = &cobra.Command{
	Use:   "affected-users",
	Short: "List users affected by a fault",
	Long:  `List the users a specific fault has affected (at most 500, those with the most notices).`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&faultsProjectID); err != nil {
			return err
		}
		if faultID == "" {
			return fmt.Errorf("fault ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		users, err := client.Faults.AffectedUsers(context.Background(), faultsProjectID, faultID,
			apiv3.Search(faultAffectedUserQuery))
		if err != nil {
			return dataAPIError("list affected users", err)
		}

		if faultOutputFormat == "json" {
			return printJSON(users)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "USER\tOCCURRENCES")
		for _, user := range users {
			_, _ = fmt.Fprintf(w, "%s\t%d\n", user.User, user.Count)
		}
		return w.Flush()
	},
}

// faultBulkChange is one of v3's bulk fault state changes.
type faultBulkChange func(
	*apiv3.FaultsService,
	context.Context,
	string,
	apiv3.FaultSelection,
) (*apiv3.FaultBulkResult, error)

// faultTimeFilter is a flag that narrows --query or --all, with the selection
// method it sets.
type faultTimeFilter struct {
	flag  string
	value *string
	apply func(apiv3.FaultSelection, time.Time) apiv3.FaultSelection
}

var faultTimeFilters = []faultTimeFilter{
	{"created-after", &faultCreatedAfter, apiv3.FaultSelection.CreatedAfter},
	{"occurred-after", &faultOccurredAfter, apiv3.FaultSelection.OccurredAfter},
	{"occurred-before", &faultOccurredBefore, apiv3.FaultSelection.OccurredBefore},
}

// faultSelection builds the selection a bulk command's flags describe, with
// its time filters and --dry-run.
func faultSelection(cmd *cobra.Command) (apiv3.FaultSelection, error) {
	sel, err := faultSelector(cmd)
	if err != nil {
		return sel, err
	}
	for _, f := range faultTimeFilters {
		if !cmd.Flags().Changed(f.flag) {
			continue
		}
		t, err := parseTimeFlag(*f.value)
		if err != nil {
			return sel, fmt.Errorf("invalid --%s: %w", f.flag, err)
		}
		// apiv3 sends Unix seconds and treats 0 as unset, so the epoch would
		// silently drop the filter (and --all would then match every fault).
		if t.Unix() <= 0 {
			return sel, fmt.Errorf("invalid --%s: must be after 1970-01-01", f.flag)
		}
		sel = f.apply(sel, t)
	}
	if faultDryRun {
		sel = sel.DryRun()
	}
	return sel, nil
}

// faultSelector picks the faults: exactly one of --id, --query or --all, with
// time filters only on --query or --all. --all needs a time filter, so no single
// command changes every fault in the project.
func faultSelector(cmd *cobra.Command) (apiv3.FaultSelection, error) {
	byID := cmd.Flags().Changed("id")
	byQuery := cmd.Flags().Changed("query")
	selectors := 0
	for _, set := range []bool{byID, byQuery, faultSelectAll} {
		if set {
			selectors++
		}
	}
	if selectors != 1 {
		return apiv3.FaultSelection{}, errors.New("use exactly one of --id, --query or --all")
	}

	filtered := slices.ContainsFunc(faultTimeFilters, func(f faultTimeFilter) bool {
		return cmd.Flags().Changed(f.flag)
	})

	var sel apiv3.FaultSelection
	switch {
	case byID:
		if filtered {
			return sel, errors.New(
				"--created-after, --occurred-after and --occurred-before apply to --query or --all, not --id",
			)
		}
		sel = apiv3.SelectFaults(faultIDs...)
	case byQuery:
		sel = apiv3.SelectFaultsMatching(faultQuery)
	default:
		if !filtered {
			return sel, errors.New(
				"--all needs --created-after, --occurred-after or --occurred-before; it won't change every fault in the project at once",
			)
		}
		sel = apiv3.SelectAllFaults()
	}
	return sel, nil
}

// pluralFaults is "1 fault" or "n faults".
func pluralFaults(n int) string {
	if n == 1 {
		return "1 fault"
	}
	return fmt.Sprintf("%d faults", n)
}

// newFaultsBulkCmd builds one of the bulk state commands (resolve, unresolve,
// ignore, unignore), which share their flags and differ only in the change.
func newFaultsBulkCmd(verb, past string, change faultBulkChange) *cobra.Command {
	title := strings.ToUpper(verb[:1]) + verb[1:]
	cmd := &cobra.Command{
		Use:   verb,
		Short: title + " faults",
		// A stray argument (an unquoted --query "a b") would otherwise be
		// dropped silently before a change to many faults.
		Args: cobra.NoArgs,
		Long: fmt.Sprintf(`%s faults in one request.

Choose the faults with exactly one of:
  --id       the named faults (repeat the flag for more than one)
  --query    every fault matching a search, as in faults list
  --all      every fault in the project that the time filters match

--created-after, --occurred-after and --occurred-before narrow --query or --all;
they can't be used with --id. --all needs a time filter, so one command can't
change every fault in the project at once. Times are YYYY-MM-DD or RFC3339.

--dry-run reports what would change without changing anything.

Examples:
  hb faults %[2]s --project-id Xk9mZp --id Fk7pQ2 --id Fk8qR3
  hb faults %[2]s --project-id Xk9mZp --query "class:RuntimeError" --dry-run
  hb faults %[2]s --project-id Xk9mZp --all --occurred-before 2026-01-01`,
			title, verb),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := resolveDataAPIProjectID(&faultsProjectID); err != nil {
				return err
			}
			sel, err := faultSelection(cmd)
			if err != nil {
				return err
			}

			client, err := newDataAPIClient()
			if err != nil {
				return err
			}

			result, err := change(client.Faults, context.Background(), faultsProjectID, sel)
			if err != nil {
				return dataAPIError(verb+" faults", err)
			}

			if faultOutputFormat == "json" {
				return printJSON(result)
			}

			summary := past + " " + pluralFaults(result.Count)
			if result.DryRun {
				summary = "Dry run, nothing changed. Would " + verb + " " + pluralFaults(
					result.Count,
				)
			}
			if len(result.FaultIds) == 0 {
				fmt.Println(summary)
				return nil
			}
			fmt.Println(summary + ":")
			for _, id := range result.FaultIds {
				fmt.Printf("  %s\n", id)
			}
			if more := result.Count - len(result.FaultIds); more > 0 {
				fmt.Printf("  …and %d more\n", more)
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&faultIDs, "id", nil, "Fault ID (repeatable)")
	cmd.Flags().StringVar(&faultQuery, "query", "", "Search query selecting the faults")
	cmd.Flags().BoolVar(&faultSelectAll, "all", false,
		"Every fault the time filters match (needs a time filter)")
	cmd.Flags().StringVar(&faultCreatedAfter, "created-after", "",
		"Only faults first seen after this time (with --query or --all)")
	cmd.Flags().StringVar(&faultOccurredAfter, "occurred-after", "",
		"Only faults that occurred after this time (with --query or --all)")
	cmd.Flags().StringVar(&faultOccurredBefore, "occurred-before", "",
		"Only faults that haven't occurred since this time (with --query or --all)")
	cmd.Flags().BoolVar(&faultDryRun, "dry-run", false,
		"Report what would change without changing anything")
	cmd.Flags().
		StringVarP(&faultOutputFormat, "output", "o", "text", "Output format (text or json)")
	return cmd
}

var (
	faultsResolveCmd = newFaultsBulkCmd("resolve", "Resolved",
		(*apiv3.FaultsService).Resolve)
	faultsUnresolveCmd = newFaultsBulkCmd("unresolve", "Unresolved",
		(*apiv3.FaultsService).Unresolve)
	faultsIgnoreCmd = newFaultsBulkCmd("ignore", "Ignored",
		(*apiv3.FaultsService).Ignore)
	faultsUnignoreCmd = newFaultsBulkCmd("unignore", "Unignored",
		(*apiv3.FaultsService).Unignore)
)

var faultMergeInto string

// faultsMergeCmd represents the faults merge command
var faultsMergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge a fault into another",
	Long: `Merge the fault named by --id into the fault named by --into.

The --id fault is removed: its notices, comments and history move to the --into
fault, and its tags, assignee and resolved or ignored state are dropped. The merge
runs in the background. This action cannot be undone.

Example:
  hb faults merge --project-id Xk9mZp --id Fk7pQ2 --into Fk8qR3`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&faultsProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		merge, err := client.Faults.Merge(
			context.Background(),
			faultsProjectID,
			faultID,
			faultMergeInto,
		)
		if err != nil {
			return dataAPIError("merge fault", err)
		}

		fmt.Printf("Merging fault %s into %s (batch %s). The merge runs in the background.\n",
			merge.SourceId, merge.TargetId, merge.BatchId)
		return nil
	},
}

var faultPauseFor string

// faultsPauseRecordingCmd represents the faults pause-recording command
var faultsPauseRecordingCmd = &cobra.Command{
	Use:   "pause-recording",
	Short: "Stop recording a fault's notices for a while",
	Long: `Stop recording new notices for a fault for an hour, a day or a week. Notices
during the pause are discarded. Pausing again replaces the end time.

Example:
  hb faults pause-recording --project-id Xk9mZp --id Fk7pQ2 --for day`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&faultsProjectID); err != nil {
			return err
		}
		duration := apiv3.PauseDuration(faultPauseFor)
		if !duration.Valid() {
			return fmt.Errorf("invalid --for %q: use hour, day or week", faultPauseFor)
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		fault, err := client.Faults.PauseRecording(
			context.Background(),
			faultsProjectID,
			faultID,
			duration,
		)
		if err != nil {
			return dataAPIError("pause recording", err)
		}

		if until, err := fault.RecordingPausedUntil.Get(); err == nil {
			fmt.Printf("Recording paused for fault %s until %s\n",
				fault.Id, until.Format("2006-01-02 15:04:05 MST"))
			return nil
		}
		fmt.Printf("Recording paused for fault %s\n", fault.Id)
		return nil
	},
}

// faultsResumeRecordingCmd represents the faults resume-recording command
var faultsResumeRecordingCmd = &cobra.Command{
	Use:   "resume-recording",
	Short: "Start recording a fault's notices again",
	Long:  `Start recording new notices for a fault whose recording was paused.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&faultsProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		fault, err := client.Faults.ResumeRecording(
			context.Background(),
			faultsProjectID,
			faultID,
		)
		if err != nil {
			return dataAPIError("resume recording", err)
		}

		fmt.Printf("Recording resumed for fault %s\n", fault.Id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(faultsCmd)
	faultsCmd.AddCommand(faultsListCmd)
	faultsCmd.AddCommand(faultsGetCmd)
	faultsCmd.AddCommand(faultsUpdateCmd)
	faultsCmd.AddCommand(faultsNoticesCmd)
	faultsCmd.AddCommand(faultsCountsCmd)
	faultsCmd.AddCommand(faultsAffectedUsersCmd)
	faultsCmd.AddCommand(faultsResolveCmd)
	faultsCmd.AddCommand(faultsUnresolveCmd)
	faultsCmd.AddCommand(faultsIgnoreCmd)
	faultsCmd.AddCommand(faultsUnignoreCmd)

	// Common flags
	faultsCmd.PersistentFlags().StringVar(&faultsProjectID, "project-id", "", "Project ID")

	// Flags for list command
	faultsListCmd.Flags().StringVarP(&faultQuery, "query", "q", "", "Search query to filter faults")
	faultsListCmd.Flags().
		StringVar(&faultOrder, "order", "recent", "Order faults by 'recent' or 'frequent'")
	faultsListCmd.Flags().
		IntVar(&faultLimit, "limit", 25, "Maximum number of faults to return (max 100)")
	faultsListCmd.Flags().
		StringVarP(&faultOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	faultsGetCmd.Flags().StringVar(&faultID, "id", "", "Fault ID")
	faultsGetCmd.Flags().
		StringVarP(&faultOutputFormat, "output", "o", "text", "Output format (text or json)")

	// Flags for update command
	faultsUpdateCmd.Flags().StringVar(&faultID, "id", "", "Fault ID")
	faultsUpdateCmd.Flags().
		BoolVar(&faultResolved, "resolved", false, "Set the fault's resolved state (use --resolved=false to un-resolve)")
	faultsUpdateCmd.Flags().
		BoolVar(&faultIgnored, "ignored", false, "Set the fault's ignored state (use --ignored=false to un-ignore)")
	faultsUpdateCmd.Flags().
		BoolVar(&faultResolveOnDeploy, "resolve-on-deploy", false, "Mark the fault to be resolved automatically on the next deploy")
	faultsUpdateCmd.Flags().
		StringVar(&faultAssigneeID, "assignee-id", "", "User ID to assign the fault to")
	faultsUpdateCmd.Flags().
		BoolVar(&faultUnassign, "unassign", false, "Remove the fault's assignee")

	// Flags for notices command
	faultsNoticesCmd.Flags().StringVar(&faultID, "id", "", "Fault ID")
	faultsNoticesCmd.Flags().
		IntVar(&faultLimit, "limit", 25, "Maximum number of notices to return (max 100)")
	faultsNoticesCmd.Flags().
		StringVarP(&faultOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for counts command
	faultsCountsCmd.Flags().
		StringVarP(&faultOutputFormat, "output", "o", "text", "Output format (text or json)")

	// Flags for affected-users command
	faultsAffectedUsersCmd.Flags().StringVar(&faultID, "id", "", "Fault ID")
	faultsAffectedUsersCmd.Flags().
		StringVarP(&faultAffectedUserQuery, "query", "q", "", "Search query to filter users")
	faultsAffectedUsersCmd.Flags().
		StringVarP(&faultOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Mark required flags
	if err := faultsGetCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}
	if err := faultsUpdateCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}
	if err := faultsNoticesCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}
	if err := faultsAffectedUsersCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}

	// merge command
	faultsCmd.AddCommand(faultsMergeCmd)
	faultsMergeCmd.Flags().StringVar(&faultID, "id", "", "ID of the fault to merge away")
	faultsMergeCmd.Flags().StringVar(&faultMergeInto, "into", "", "ID of the fault to keep")
	_ = faultsMergeCmd.MarkFlagRequired("id")
	_ = faultsMergeCmd.MarkFlagRequired("into")

	// pause-recording and resume-recording commands
	faultsCmd.AddCommand(faultsPauseRecordingCmd)
	faultsCmd.AddCommand(faultsResumeRecordingCmd)
	faultsPauseRecordingCmd.Flags().StringVar(&faultID, "id", "", "Fault ID")
	faultsPauseRecordingCmd.Flags().
		StringVar(&faultPauseFor, "for", "", "How long to pause: hour, day or week")
	_ = faultsPauseRecordingCmd.MarkFlagRequired("id")
	_ = faultsPauseRecordingCmd.MarkFlagRequired("for")
	faultsResumeRecordingCmd.Flags().StringVar(&faultID, "id", "", "Fault ID")
	_ = faultsResumeRecordingCmd.MarkFlagRequired("id")
}
