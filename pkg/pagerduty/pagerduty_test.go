package pagerduty

import (
	"sort"
	"testing"
)

// Schedule names as PagerDuty renders them since the escalation-ladder rework:
// one primary schedule per team/shift, plus one copy per team member.
var schedules = map[string]string{
	"P1": "phoenix_afternoon On-Call Schedule",
	"P2": "phoenix_afternoon On-Call Schedule (1)",
	"P3": "phoenix_afternoon On-Call Schedule (7)",
	"P4": "phoenix_morning On-Call Schedule",
	"P5": "phoenix_morning On-Call Schedule (1)",
	"P6": "atlas On-Call Schedule",
	"P7": "atlas On-Call Schedule (3)",
	"P8": "atlas On-Call Schedule (Catchup)",
	"P9": "phoenixdev_afternoon On-Call Schedule",
}

func TestMatchSchedules(t *testing.T) {
	for _, tc := range []struct {
		name    string
		team    string
		morning bool
		want    []string
	}{
		{"split schedule, afternoon shift", "team-phoenix", false, []string{"P1"}},
		{"split schedule, morning shift", "team-phoenix", true, []string{"P4"}},
		{"single schedule", "team-atlas", false, []string{"P6"}},
		{"team without a schedule", "team-nosuch", false, []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := matchSchedules(schedules, tc.team, tc.morning)
			sort.Strings(got)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}
