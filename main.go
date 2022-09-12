package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	TEAM_PATTERN       = `^team-[a-z0-9]*$`
	GITHUB_URL_PATTERN = `^.*/github\.com/([^/]*).*$`
)

func load() *Config {
	var (
		configFile string
		err        error
		cfg        *Config
	)
	flag.StringVar(&configFile, "config", os.Getenv("AILE_CONFIG_FILE"),
		"Config filename - can also be set by environment variable `AILE_CONFIG_FILE`")
	flag.Parse()

	if configFile == "" {
		err = fmt.Errorf("Missing configfile - please specify either AILE_CONFIG_FILE env var or -config")
	} else if _, err := os.Stat(configFile); err != nil {
		fmt.Errorf("Config file does not exist or is unreadable")
	} else if cfg, err = NewConfig(configFile); err != nil {
		err = err
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return cfg
}

func parseTeamMembers(team *Team, slackUsers []Member, afkEvents []string) {
	for _, u := range team.Members {
		for _, i := range slackUsers {
			if i.GithubLogin == u.GithubLogin {
				u.SlackID = i.SlackID
				u.Email = i.Email
				u.Afk = containsString(i.Email, afkEvents)
			}
		}
	}
}

func main() {
	var cfg *Config = load()

	g := NewGithub(cfg)
	s := NewSlack(cfg.SlackToken, cfg.PagingEntries)
	o := NewOpsGenie(cfg.OpsGenieToken)
	c := NewCalendar(cfg)

	var (
		afkEvents  []string            = c.CurrentShiftEvents()
		slackUsers []Member            = s.Users(cfg.Domain)
		topics     map[string][]string = s.Topics(TEAM_PATTERN)
		teams      []*Team             = make([]*Team, 0)
	)

	for _, t := range g.Teams(cfg.Organisation, TEAM_PATTERN) {
		if !t.IsEngineeringTeam() && !t.IsAccountEngineering() {
			continue
		}
		if topic, ok := topics[t.Name]; ok {
			t.Topics = topic
		}
		go parseTeamMembers(t, slackUsers, afkEvents)
		o.WhoIsOnCall(t)
		teams = append(teams, t)
	}

	s.SlackHandles(teams)
}
