package main

import (
	"flag"
	"fmt"
	"log"
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
		err = fmt.Errorf("Config file does not exist or is unreadable")
	} else if cfg, err = NewConfig(configFile); err != nil {
		//noop - just about assignment
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
	c := NewCalendar(cfg)
	o := NewOpsGenie(cfg.OpsGenieToken, c)

	var (
		afkEvents  []string
		slackUsers []Member
		topchan    chan map[string][]string = make(chan map[string][]string)
		topics     map[string][]string
		teams      []*Team       = make([]*Team, 0)
		teamchan   chan []*Team  = make(chan []*Team)
		calchan    chan []string = make(chan []string)
	)
	defer close(topchan)
	defer close(teamchan)
	defer close(calchan)

	go c.CurrentShiftEvents(&calchan)
	go g.Teams(cfg.Organisation, TEAM_PATTERN, &teamchan)

	// These next two take 4 seconds to execute
	go s.GetUsersPaginated(cfg.Domain, &slackUsers)
	go s.Topics(TEAM_PATTERN, &topchan)
	topics = <-topchan
	afkEvents = <-calchan

	for _, t := range <-teamchan {
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

	log.Println("Creating handles")
	s.SlackHandles(teams)
	log.Println("Done")
}
