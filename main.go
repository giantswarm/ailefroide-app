package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	aile "github.com/giantswarm/ailefroide/pkg/ailefroide"
	ac "github.com/giantswarm/ailefroide/pkg/calendar"
	ag "github.com/giantswarm/ailefroide/pkg/github"
	ao "github.com/giantswarm/ailefroide/pkg/opsgenie"
	as "github.com/giantswarm/ailefroide/pkg/slack"
)

const (
	TEAM_PATTERN       = `^team-[a-z0-9]*$`
	SUPPORT_PATTERN    = `^support-[a-z0-9]+(-[\w]+)?$`
	GITHUB_URL_PATTERN = `^.*/github\.com/([^/]*).*$`
)

func load() *aile.Config {
	var (
		configFile string
		err        error
		cfg        *aile.Config
	)
	flag.StringVar(&configFile, "config", os.Getenv("AILE_CONFIG_FILE"),
		"Config filename - can also be set by environment variable `AILE_CONFIG_FILE`")
	flag.Parse()

	if configFile == "" {
		err = fmt.Errorf("Missing configfile - please specify either AILE_CONFIG_FILE env var or -config")
	} else if _, err = os.Stat(configFile); err != nil {
		err = fmt.Errorf("Config file does not exist or is unreadable")
	} else if cfg, err = aile.NewConfig(configFile); err != nil {
		//noop - just about assignment
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return cfg
}

func parseTeamMembers(team *aile.Team, slackUsers []aile.Member, afkEvents []string) {
	for _, u := range team.Members {
		for _, i := range slackUsers {
			if i.GithubLogin == u.GithubLogin {
				u.SlackID = i.SlackID
				u.Email = i.Email
				u.Afk = aile.ContainsString(i.Email, afkEvents)
			}
		}
	}
}

func main() {
	var cfg *aile.Config = load()

	g := ag.NewGithub(cfg)
	s := as.NewSlack(cfg.SlackToken, SUPPORT_PATTERN, cfg.PagingEntries)
	c := ac.NewCalendar(cfg)
	o := ao.NewOpsGenie(cfg.OpsGenieToken, c)

	var (
		afkEvents  []string
		slackUsers []aile.Member
		topchan    chan map[string][]string = make(chan map[string][]string)
		topics     map[string][]string
		teams      []*aile.Team       = make([]*aile.Team, 0)
		teamchan   chan []*aile.Team  = make(chan []*aile.Team)
		calchan    chan []string      = make(chan []string)
		userchan   chan []aile.Member = make(chan []aile.Member)
	)
	defer close(topchan)
	defer close(teamchan)
	defer close(calchan)
	defer close(userchan)

	go c.CurrentShiftEvents(&calchan)
	go g.Teams(cfg.Organisation, TEAM_PATTERN, &teamchan)

	go s.Topics(TEAM_PATTERN, &topchan)
	go s.GetUsersPaginated(cfg.Domain, GITHUB_URL_PATTERN, &userchan)

	slackUsers = <-userchan
	topics = <-topchan
	afkEvents = <-calchan

	for _, t := range <-teamchan {
		if !t.IsEngineeringTeam() && !t.IsAccountEngineering() {
			continue
		}
		if topic, ok := topics[t.Name]; ok {
			t.Topics = topic
		}
		parseTeamMembers(t, slackUsers, afkEvents)
		o.WhoIsOnCall(t)
		teams = append(teams, t)
	}

	log.Println("Creating handles")
	s.SlackHandles(teams)
	log.Println("Done")
}
