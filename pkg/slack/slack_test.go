package slack

import (
	"reflect"
	"testing"

	aile "github.com/giantswarm/ailefroide/pkg/ailefroide"
)

const productOwner = "UPO"

// A team without a solution architect: its handle holds the product owner,
// plus whoever WhoIsOnCall marked as on call for the current run.
func TestSupportMembers(t *testing.T) {
	settings := aile.TeamSettings{
		IncludeProductOwner:            true,
		IncludeOnCallEngineer:          true,
		RotateWithoutSolutionArchitect: true,
	}

	team := func(oncall string) *aile.Team {
		members := []*aile.Member{
			{Email: "po@example.com", SlackID: productOwner, IsProductOwner: true},
			{Email: "first@example.com", SlackID: "U1"},
			{Email: "second@example.com", SlackID: "U2"},
		}
		for _, m := range members {
			m.Oncall = m.SlackID == oncall
		}
		return &aile.Team{Name: "team-x", Members: members}
	}

	for _, tc := range []struct {
		name     string
		team     *aile.Team
		settings aile.TeamSettings
		want     []string
	}{
		{"on call during business hours", team("U1"), settings, []string{productOwner, "U1"}},
		{"shift over, nobody on call", team(""), settings, []string{productOwner}},
		{"next day's shift", team("U2"), settings, []string{productOwner, "U2"}},
		{"on call, but includeOCE off", team("U1"), aile.TeamSettings{IncludeProductOwner: true}, []string{productOwner}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := supportMembers(tc.team, tc.settings, false); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("supportMembers() = %v, want %v", got, tc.want)
			}
		})
	}
}

// An on-call engineer who is AFK in Personio still covers their shift.
func TestSupportMembersOncallOverridesAfk(t *testing.T) {
	team := &aile.Team{Name: "team-x", Members: []*aile.Member{
		{Email: "first@example.com", SlackID: "U1", Oncall: true, Afk: true},
	}}
	got := supportMembers(team, aile.TeamSettings{IncludeOnCallEngineer: true}, false)
	if want := []string{"U1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("supportMembers() = %v, want %v", got, want)
	}
}
