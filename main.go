package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	aile "github.com/giantswarm/ailefroide/pkg/ailefroide"
	ac "github.com/giantswarm/ailefroide/pkg/calendar"
	ag "github.com/giantswarm/ailefroide/pkg/github"
	ao "github.com/giantswarm/ailefroide/pkg/opsgenie"
	as "github.com/giantswarm/ailefroide/pkg/slack"
	ap "github.com/giantswarm/personio-go/v1"
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

		var include = false
		for _, n := range ec {
			if strings.EqualFold(team.Members[k].Email, n) || strings.EqualFold(team.Members[k].GithubLogin, n) {
				include = true
				break
			}
		}

		var login string = team.Members[k].GithubLogin
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

func main() {
	var (
		cfg    *aile.Config = load()
		people []*ap.Employee
		e      error
		creds  ap.Credentials = ap.Credentials{
			ClientId:     cfg.PersonioClientId,
			ClientSecret: cfg.PersonioClientSecret,
		}
		p *ap.Client
	)

	//cfg.PersonioGHFieldId
	if p, e = ap.NewClient(context.TODO(), PERSONIO_API, creds); e == nil {
		people, _ = p.GetEmployees()
	}

	g := ag.NewGithub(cfg, people)
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

	go g.Teams(cfg.Organisation, TEAM_PATTERN, &teamchan)
	go s.Topics(TEAM_PATTERN, &topchan)
	go s.GetPersonioUsersPaginated(cfg.Domain, people, &userchan, cfg.PersonioGHFieldId)

	slackUsers = <-userchan
	topics = <-topchan
	var (
		start, end time.Time = c.CurrentShift()
		absences   []*ap.TimeOff
	)

	absences, _ = p.GetTimeOffs(&start, &end, 0, 1000)

	for _, t := range absences {
		var (
			isFullDay   bool = !bool(t.HalfDayStart) && !bool(t.HalfDayEnd)
			isMorning   bool = bool(t.HalfDayStart) && c.IsMorning()
			isAfternoon bool = bool(t.HalfDayEnd) && !c.IsMorning()
		)

		if isFullDay || isMorning || isAfternoon {
			afkEvents = append(afkEvents, *t.Employee.GetStringAttribute("email"))
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
