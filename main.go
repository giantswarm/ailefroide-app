package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	g := NewGithub(os.Getenv("GITHUB_TOKEN"))
	s := NewSlack(os.Getenv("SLACK_TOKEN"))
	o := NewOpsGenie(os.Getenv("OPSGENIE_TOKEN"))
	o.ListSchedules()
	slackUsers := s.Users()
	topics := s.Topics(TEAM_PATTERN)
	for _, t := range g.Teams(ORGANISATION, TEAM_PATTERN) {
		if !t.IsEngineeringTeam() && !t.IsAccountEngineering() {
			continue
		}
		if topic, ok := topics[t.Name]; ok {
			t.Topics = topic
		}
		fmt.Println(t.Name)
		fmt.Println(strings.Join(t.Topics, " | "))
		for _, u := range t.Members {
			for _, i := range slackUsers {
				if i.GithubLogin == u.GithubLogin {
					u.SlackID = i.SlackID
					u.Email = i.Email
				}
			}
		}
		o.WhoIsOnCall(t)

		for _, u := range t.Members {
			fmt.Printf("  - %+v\n", u)
		}
	}
}
