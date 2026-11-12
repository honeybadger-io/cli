package cmd

import (
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// v3ReferencePages is the v3 Data API reference page each Data API command's
// help points at: the page documents the JSON the command takes with
// --cli-input-json and prints with --output json.
var v3ReferencePages = map[string][]string{
	"alarms list":             {"alarms/list-alarms"},
	"alarms get":              {"alarms/get-alarm"},
	"alarms create":           {"alarms/create-alarm"},
	"alarms update":           {"alarms/update-alarm"},
	"alarms delete":           {"alarms/delete-alarm"},
	"alarms history":          {"alarms/list-alarm-history"},
	"check-ins list":          {"check-ins/list-check-ins"},
	"check-ins get":           {"check-ins/get-check-in"},
	"check-ins create":        {"check-ins/create-check-in"},
	"check-ins update":        {"check-ins/update-check-in"},
	"check-ins delete":        {"check-ins/delete-check-in"},
	"check-ins events":        {"check-ins/list-check-in-events"},
	"comments list":           {"comments/list-comments"},
	"comments get":            {"comments/get-comment"},
	"comments create":         {"comments/create-comment"},
	"comments update":         {"comments/update-comment"},
	"comments delete":         {"comments/delete-comment"},
	"dashboards list":         {"dashboards/list-dashboards"},
	"dashboards get":          {"dashboards/get-dashboard"},
	"dashboards create":       {"dashboards/create-dashboard"},
	"dashboards update":       {"dashboards/update-dashboard"},
	"dashboards delete":       {"dashboards/delete-dashboard"},
	"deployments list":        {"deploys/list-deploys"},
	"deployments get":         {"deploys/get-deploy"},
	"environments list":       {"environments/list-environments"},
	"environments get":        {"environments/get-environment"},
	"environments create":     {"environments/create-environment"},
	"environments update":     {"environments/update-environment"},
	"environments delete":     {"environments/delete-environment"},
	"faults list":             {"faults/list-faults"},
	"faults get":              {"faults/get-fault"},
	"faults notices":          {"notices/list-notices"},
	"faults update":           {"faults/update-fault"},
	"faults counts":           {"faults/get-fault-summary"},
	"faults affected-users":   {"faults/list-fault-affected-users"},
	"faults resolve":          {"faults/resolve-faults"},
	"faults unresolve":        {"faults/unresolve-faults"},
	"faults ignore":           {"faults/ignore-faults"},
	"faults unignore":         {"faults/unignore-faults"},
	"faults merge":            {"faults/merge-faults"},
	"faults pause-recording":  {"faults/pause-fault-recording"},
	"faults resume-recording": {"faults/resume-fault-recording"},
	"faults delete":           {"faults/delete-fault"},
	"ingestion-keys list":     {"ingestion-keys/list-ingestion-keys"},
	"ingestion-keys get":      {"ingestion-keys/get-ingestion-key"},
	"ingestion-keys create":   {"ingestion-keys/create-ingestion-key"},
	"ingestion-keys update":   {"ingestion-keys/update-ingestion-key"},
	"ingestion-keys delete":   {"ingestion-keys/delete-ingestion-key"},
	"insights query":          {"insights/run-insights-query"},
	"integrations list":       {"integrations/list-integrations"},
	"integrations get":        {"integrations/get-integration"},
	"integrations create":     {"integrations/create-integration"},
	"integrations update":     {"integrations/update-integration"},
	"integrations delete":     {"integrations/delete-integration"},
	"projects list":           {"projects/list-projects"},
	"projects get":            {"projects/get-project"},
	"projects create":         {"projects/create-project"},
	"projects update":         {"projects/update-project"},
	"projects delete":         {"projects/delete-project"},
	"projects occurrences": {
		"projects/get-project-occurrences",
		"projects/list-account-occurrences",
	},
	"statuspages list":    {"status-pages/list-status-pages"},
	"statuspages get":     {"status-pages/get-status-page"},
	"statuspages create":  {"status-pages/create-status-page"},
	"statuspages update":  {"status-pages/update-status-page"},
	"statuspages delete":  {"status-pages/delete-status-page"},
	"streams list":        {"streams/list-streams"},
	"teams list":          {"teams/list-teams"},
	"teams get":           {"teams/get-team"},
	"teams create":        {"teams/create-team"},
	"teams update":        {"teams/update-team"},
	"teams delete":        {"teams/delete-team"},
	"uptime sites list":   {"uptime-sites/list-sites"},
	"uptime sites get":    {"uptime-sites/get-site"},
	"uptime sites create": {"uptime-sites/create-site"},
	"uptime sites update": {"uptime-sites/update-site"},
	"uptime sites delete": {"uptime-sites/delete-site"},
	"uptime outages":      {"uptime-sites/list-outages"},
	"uptime checks":       {"uptime-sites/list-uptime-checks"},
	"whoami":              {"tokens/get-token"},
}

var referenceLine = regexp.MustCompile(
	`(?m)^Reference: https://docs\.honeybadger\.io/api/v3/([a-z-]+/[a-z-]+)/$`,
)

// dataAPILeaves returns every runnable command under the Data API group, keyed
// by its path without the leading "hb ".
func dataAPILeaves() map[string]*cobra.Command {
	leaves := map[string]*cobra.Command{}
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		if c.Runnable() {
			leaves[strings.TrimPrefix(c.CommandPath(), rootCmd.Name()+" ")] = c
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	for _, c := range rootCmd.Commands() {
		if c.GroupID == GroupDataAPI {
			walk(c)
		}
	}
	return leaves
}

func TestEveryDataAPICommandLinksItsV3Reference(t *testing.T) {
	leaves := dataAPILeaves()
	for path, c := range leaves {
		want, known := v3ReferencePages[path]
		if !assert.True(t, known, "%q has no entry in v3ReferencePages", path) {
			continue
		}
		var got []string
		for _, m := range referenceLine.FindAllStringSubmatch(c.Long, -1) {
			got = append(got, m[1])
		}
		assert.Equal(t, want, got, "%q help reference pages", path)
		assert.True(t, strings.HasSuffix(c.Long, "/"),
			"%q help should end with its reference line", path)
	}
	for path := range v3ReferencePages {
		assert.Contains(t, leaves, path, "v3ReferencePages lists %q, which isn't a command", path)
	}
}

func TestReportingAPICommandsHaveNoV3Reference(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.GroupID == GroupReportingAPI {
			assert.NotRegexp(t, referenceLine, c.Long, c.Name())
		}
	}
}
