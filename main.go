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
	ap "github.com/giantswarm/ailefroide/pkg/personio"
	as "github.com/giantswarm/ailefroide/pkg/slack"
)

// Matching patterns for team and support handles
// lint:ignore noallcaps
const (
	TEAM_PATTERN    = `^team-[a-z0-9]*$`
	SUPPORT_PATTERN = `^support-[a-z0-9]+(-[\w]+)?$`
)

func load() *aile.Config {
	var (
		configFile string
		debug      bool
		debugTeam  string
		err        error
		cfg        *aile.Config
	)
	flag.StringVar(&configFile, "config", os.Getenv("AILE_CONFIG_FILE"),
		"Config filename - can also be set by environment variable `AILE_CONFIG_FILE`")
	flag.BoolVar(&debug, "debug", false, "Turn on debugging in the application")
	flag.StringVar(&debugTeam, "debugteam", "", "Team name for debugging purposes - required if debug is true")
	flag.Parse()

	if configFile == "" {
		err = fmt.Errorf("Missing configfile - please specify either AILE_CONFIG_FILE env var or -config")
	} else if _, err = os.Stat(configFile); err != nil {
		err = fmt.Errorf("Config file does not exist or is unreadable")
	} else if cfg, err = aile.NewConfig(configFile); err != nil {
		log.Println("Config loaded")
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	cfg.Debug = debug
	cfg.DebugTeam = debugTeam
	return cfg
}

func parseTeamMembers(team *aile.Team, slackUsers []aile.Member, afkEvents []string, g *ag.Github) {
	for k, u := range team.Members {
		for s, i := range slackUsers {
			if u.GithubLogin != "" && (i.GithubLogin == u.GithubLogin) {
				u.SlackID = i.SlackID
				u.Email = i.Email
				team.Members[k] = u
			} else if i.Email != "" && (i.Email == u.Email) {
				team.Members[k] = &slackUsers[s]
			}

			var login string = team.Members[k].GithubLogin
			team.Members[k].IsAccountEngineer = g.IsAccountEngineer(login)
			team.Members[k].IsProductOwner = g.IsProductOwner(login)
			team.Members[k].IsSolutionArchitect = g.IsSolutionArchitect(login)
			team.Members[k].IsPlatformArchitect = g.IsPlatformArchitect(login)
			team.Members[k].IsSiteReliabilityEngineer = g.IsSiteReliabilityEngineer(login)
			team.Members[k].Afk = aile.ContainsString(team.Members[k].Email, afkEvents)
		}
	}
}

func main() {
	var (
		cfg    *aile.Config = load()
		people []ap.Employee
	)

	if p, e := ap.New(cfg.PersonioClientId, cfg.PersonioClientSecret, cfg.PersonioGHFieldId); e == nil {
		people, _ = p.Employees()
	}

	g := ag.NewGithub(cfg)
	s := as.NewSlack(cfg.SlackToken, SUPPORT_PATTERN, cfg.PagingEntries, cfg.Teams)
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
	go s.GetPersonioUsersPaginated(cfg.Domain, people, &userchan)

	slackUsers = <-userchan
	topics = <-topchan
	afkEvents = <-calchan

	for _, t := range <-teamchan {
		parseTeamMembers(t, slackUsers, afkEvents, g)
		if !t.IsEngineeringTeam() && !t.IsAccountEngineering() {
			continue
		}
		if topic, ok := topics[t.Name]; ok {
			t.Topics = topic
		}
		o.WhoIsOnCall(t)
		teams = append(teams, t)
	}

	log.Println("Creating handles")
	s.SlackHandles(teams, cfg.Debug, cfg.DebugTeam)
	log.Println("Done")
}
