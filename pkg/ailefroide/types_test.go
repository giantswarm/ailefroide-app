package ailefroide

import "testing"

const teamName = "team-x"

func TestInRotation(t *testing.T) {
	var (
		engineer  = &Member{GithubLogin: "engineer"}
		architect = &Member{GithubLogin: "architect", IsSolutionArchitect: true}
		accounts  = &Member{GithubLogin: "accounts", IsAccountEngineer: true}
		optIn     = TeamSettings{RotateWithoutSolutionArchitect: true}
	)

	for _, tc := range []struct {
		name     string
		members  []*Member
		settings map[string]TeamSettings
		want     bool
	}{
		{"opted in, no solution architect", []*Member{engineer}, map[string]TeamSettings{teamName: optIn}, true},
		{"opted in but skipped", []*Member{engineer}, map[string]TeamSettings{teamName: {SkipRotation: true, RotateWithoutSolutionArchitect: true}}, false},
		{"configured without opt-in, no solution architect", []*Member{engineer}, map[string]TeamSettings{teamName: {}}, false},
		{"configured without opt-in, with solution architect", []*Member{engineer, architect}, map[string]TeamSettings{teamName: {}}, true},
		{"another team opted in", []*Member{engineer}, map[string]TeamSettings{"team-y": optIn}, false},
		{"unconfigured, with solution architect", []*Member{engineer, architect}, map[string]TeamSettings{}, true},
		{"unconfigured, account engineering", []*Member{accounts}, map[string]TeamSettings{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			team := &Team{Name: teamName, Members: tc.members}
			if got := team.InRotation(tc.settings); got != tc.want {
				t.Errorf("InRotation() = %t, want %t", got, tc.want)
			}
		})
	}
}
