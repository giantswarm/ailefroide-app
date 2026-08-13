package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	ap "github.com/giantswarm/personio-go/v1"

	aile "github.com/giantswarm/ailefroide/pkg/ailefroide"
	ac "github.com/giantswarm/ailefroide/pkg/calendar"
	ag "github.com/giantswarm/ailefroide/pkg/github"
	ao "github.com/giantswarm/ailefroide/pkg/pagerduty"
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
		err = fmt.Errorf("missing configfile - please specify either AILE_CONFIG_FILE env var or -config")
	} else if _, err = os.Stat(configFile); err != nil {
		err = fmt.Errorf("config file does not exist or is unreadable")
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

func parseTeamMembers(team *aile.Team, slackUsers []aile.Member, afkEvents []string, g *ag.Github, ec []string) {
	for k, tm := range team.Members {
		for s, su := range slackUsers {
			if tm.GithubLogin != "" && strings.EqualFold(su.GithubLogin, tm.GithubLogin) {
				tm.SlackID = su.SlackID
				tm.Email = su.Email
				team.Members[k] = tm
			} else if su.Email != "" && strings.EqualFold(su.Email, tm.Email) {
				team.Members[k] = &slackUsers[s]
			}
		}

		include := false
		for _, n := range ec {
			if strings.EqualFold(team.Members[k].Email, n) || strings.EqualFold(team.Members[k].GithubLogin, n) {
				include = true
				break
			}
		}

		login := team.Members[k].GithubLogin
		team.Members[k].IsAccountEngineer = g.IsAccountEngineer(login)
		team.Members[k].IsProductOwner = g.IsProductOwner(login)
		team.Members[k].IsSolutionArchitect = g.IsSolutionArchitect(login)
		team.Members[k].IsPlatformArchitect = g.IsPlatformArchitect(login)
		team.Members[k].IsSiteReliabilityEngineer = g.IsSiteReliabilityEngineer(login)
		team.Members[k].Afk = aile.ContainsString(team.Members[k].Email, afkEvents)
		team.Members[k].IncludeWhenNotAFK = include
	}
}

const PERSONIO_API = "https://api.personio.de/v1"

func isBetween(t, min, max time.Time) bool {
	if min.After(max) {
		min, max = max, min
	}
	return (t.Equal(min) || t.After(min)) && (t.Equal(max) || t.Before(max))
}

func main() {
	var (
		cfg    = load()
		people []*ap.Employee
		e      error
		creds  = ap.Credentials{
			ClientId:     cfg.PersonioClientId,
			ClientSecret: cfg.PersonioClientSecret,
		}
		p *ap.Client
	)

	if p, e = ap.NewClient(context.TODO(), PERSONIO_API, creds); e == nil {
		people, _ = p.GetEmployees()
	}

	g, e := ag.NewGithub(cfg, people)
	if e != nil {
		log.Println("Error setting up Github", e)
		os.Exit(1)
	}
	s := as.NewSlack(cfg.SlackToken, SUPPORT_PATTERN, cfg.PagingEntries, cfg.Teams)
	c := ac.NewCalendar(cfg)
	o, e := ao.New(cfg.PagerDutyToken, c)
	if e != nil {
		log.Println("Error setting up PagerDuty", e)
		os.Exit(1)
	}

	var (
		afkEvents  []string
		slackUsers []aile.Member
		topchan    = make(chan map[string][]string)
		topics     map[string][]string
		teams      = make([]*aile.Team, 0)
		teamchan   = make(chan []*aile.Team)
		userchan   = make(chan []aile.Member)
	)
	defer close(topchan)
	defer close(teamchan)
	defer close(userchan)

	go g.Teams(cfg.Organisation, TEAM_PATTERN, &teamchan)
	go s.Topics(TEAM_PATTERN, &topchan)
	go s.GetPersonioUsersPaginated(cfg.Domain, people, &userchan, cfg.PersonioGHFieldId)

	slackUsers = <-userchan
	topics = <-topchan
	var (
		start, end time.Time = c.CurrentShift()
		absences   []*ap.TimeOff
		err        error
	)

	absences, err = p.GetTimeOffsMapped(start, end)
	if err != nil {
		log.Println("Error getting absences", err)
		os.Exit(1)
	}

	for _, absence := range absences {
		email := absence.Employee.GetStringAttribute("email")
		if email == nil || aile.ContainsString(*email, afkEvents) {
			continue
		}

		if isBetween(time.Now(), absence.StartDate, absence.EndDate) {
			log.Printf("Logging AFK for %s - Start %s : End %s\n", *email, absence.StartDate, absence.EndDate)
			afkEvents = append(afkEvents, *email)
		}
	}

	for _, t := range <-teamchan {
		parseTeamMembers(t, slackUsers, afkEvents, g, cfg.Teams[t.Name].ExtraCover)
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
