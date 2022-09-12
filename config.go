package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Domain             string `yaml:"domain"`
	Organisation       string `yaml:"organisation"`
	AccountEngineers   string `yaml:"accountEngineers"`
	SolutionArchitects string `yaml:"solutionArchitects"`
	ProductOwners      string `yaml:"productOwners"`
	AfkCalendar        string `yaml:"calendarId"`

	// Credentials
	CalendarTokenFile string `yaml:"calendarCredentialsFile,omitempty"`
	GithubToken       string `yaml:"githubToken,omitempty"`
	OpsGenieToken     string `yaml:"opsGenieToken,omitempty"`
	SlackToken        string `yaml:"slackToken,omitempty"`

	Location            string        `yaml:"location" default:"Europe/Berlin"`
	PagingEntries       int           `yaml:"itemsPerPage" default:"200"`
	Timeout             time.Duration `yaml:"timeout" default:"100ms"`
	CalendarCredentials []byte
}

func NewConfig(filename string) (*Config, error) {
	cfg := struct {
		Config Config `yaml:"config"`
	}{}

	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, &cfg)
	c := &cfg.Config
	defaults.Set(c)

	if gt := os.Getenv("GITHUB_TOKEN"); gt != "" {
		c.GithubToken = gt
	}
	if st := os.Getenv("SLACK_TOKEN"); st != "" {
		c.SlackToken = st
	}

	if ot := os.Getenv("OPSGENIE_TOKEN"); ot != "" {
		c.OpsGenieToken = ot
	}

	err = c.validate()
	if err == nil {
		c.CalendarCredentials, err = os.ReadFile(c.CalendarTokenFile)
		if err != nil {
			err = fmt.Errorf("Unable to read client secret file: %v", err)
		}
	}
	return c, err
}

func (c *Config) validate() error {
	var messages []string = make([]string, 0)
	if c.GithubToken == "" {
		messages = append(messages, "GITHUB_TOKEN is missing")
	}
	if c.SlackToken == "" {
		messages = append(messages, "SLACK_TOKEN is missing")
	}
	if c.OpsGenieToken == "" {
		messages = append(messages, "OPSGENIE_TOKEN is missing")
	}
	if c.CalendarTokenFile == "" {
		messages = append(messages, "Config entry CalendarTokenFile is missing")
	} else {
		if _, err := os.Stat(c.CalendarTokenFile); err != nil {
			messages = append(messages, err.Error())
		}
	}
	if len(messages) > 0 {
		return fmt.Errorf("The following required values are invalid or missing: %s", strings.Join(messages, ", "))
	}
	return nil
}
