package main

import (
	"os"
	"time"
)

const (
	DOMAIN              = "giantswarm.io"
	ORGANISATION        = "giantswarm"
	ACCOUNT_ENGINEERS   = "chapter-ae"
	SOLUTION_ARCHITECTS = "chapter-se"
	TEAM_PATTERN        = `^team-[a-z0-9]*$`
	GITHUB_URL_PATTERN  = `^.*/github\.com/([^/]*).*$`
	DURATION            = 100 * time.Millisecond
)

func main() {
	g := NewGithub(os.Getenv("GITHUB_TOKEN"))
	s := NewSlack(os.Getenv("SLACK_TOKEN"))
	o := NewOpsGenie(os.Getenv("OPSGENIE_TOKEN"))
	c := NewCalendar()

	var (
		afkEvents  []string            = c.CurrentShiftEvents()
		slackUsers []Member            = s.Users()
		topics     map[string][]string = s.Topics(TEAM_PATTERN)
		teams      []*Team             = make([]*Team, 0)
	)
	for _, t := range g.Teams(ORGANISATION, TEAM_PATTERN) {
		if !t.IsEngineeringTeam() && !t.IsAccountEngineering() {
			continue
		}
		if topic, ok := topics[t.Name]; ok {
			t.Topics = topic
		}
		for _, u := range t.Members {
			for _, i := range slackUsers {
				if i.GithubLogin == u.GithubLogin {
					u.SlackID = i.SlackID
					u.Email = i.Email
					u.Afk = containsString(i.Email, afkEvents)
				}
			}
		}
		o.WhoIsOnCall(t)
		teams = append(teams, t)
	}
	s.SlackHandles(teams)
}
