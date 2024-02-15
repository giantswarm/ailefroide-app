package ailefroide

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v2"
)

type Github struct {
	AppId          int64  `yaml:"appId"`
	InstallationId int64  `yaml:"installationId"`
	PrivateKey     string `yaml:"privateKey"`
}

type TeamSettings struct {
	SkipRotation             bool     `yaml:"skipRotation" default:"false"`
	IncludeProductOwner      bool     `yaml:"includePo" default:"true"`
	IncludeOnCallEngineer    bool     `yaml:"includeOCE" default:"true"`
	IncludePlatformArchitect bool     `yaml:"includePa" default:"false"`
	IncludeSRE               bool     `yaml:"includeSRE" default:"false"`
	ExtraCover               []string `yaml:"extraCover"`
}

type Config struct {
	Domain             string                  `yaml:"domain"`
	Organisation       string                  `yaml:"organisation"`
	AccountEngineers   string                  `yaml:"accountEngineers"`
	SolutionArchitects string                  `yaml:"solutionArchitects"`
	ProductOwners      string                  `yaml:"productOwners"`
	PlatformArchitects string                  `yaml:"platformArchitects"`
	SREs               string                  `yaml:"siteReliabilityEngineers"`
	AfkCalendar        string                  `yaml:"calendarId"`
	PersonioGHFieldId  string                  `yaml:"personioGithubFieldId"`
	Teams              map[string]TeamSettings `yaml:"teams"`

	// Credentials
	OpsGenieToken        string `yaml:"opsGenieToken,omitempty"`
	SlackToken           string `yaml:"slackToken,omitempty"`
	PersonioClientId     string `yaml:"persionioClientId"`
	PersonioClientSecret string `yaml:"persionioClientSecret"`
	Gh                   Github `yaml:"github"`

	Location      string        `yaml:"location" default:"Europe/Berlin"`
	PagingEntries int           `yaml:"itemsPerPage" default:"200"`
	Timeout       time.Duration `yaml:"timeout" default:"100ms"`

	StartOfDay 	   string `yaml:"startOfDay" default:"09:00"`
	MiddayShiftChange string `yaml:"midday" default:"13:00"`
	EndOfDay          string `yaml:"endOfDay" default:"18:00"`

	CalendarCredentials []byte
	Debug               bool
	DebugTeam           string
}

func NewConfig(filename string) (*Config, error) {
	cfg := struct {
		Config Config `yaml:"config"`
	}{}

	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
		return nil, err
	}

	if err = yaml.Unmarshal(yamlFile, &cfg); err != nil {
		log.Println("unable to read config file or file is invalid")
		return nil, err
	}

	c := &cfg.Config
	if err := defaults.Set(c); err != nil {
		log.Println("Unable to set defalt values on config")
		return nil, err
	}

	if st := os.Getenv("SLACK_TOKEN"); st != "" {
		c.SlackToken = st
	}

	if ot := os.Getenv("OPSGENIE_TOKEN"); ot != "" {
		c.OpsGenieToken = ot
	}

	if pt := os.Getenv("PERSONIO_CLIENT_ID"); pt != "" {
		c.PersonioClientId = pt
	}

	if ps := os.Getenv("PERSONIO_CLIENT_SECRET"); ps != "" {
		c.PersonioClientSecret = ps
	}

	if cfg.Config.Teams == nil {
		cfg.Config.Teams = make(map[string]TeamSettings)
	}

	err = c.validate()
	return c, err
}

func (c *Config) validate() error {
	var messages []string = make([]string, 0)
	if c.SlackToken == "" {
		messages = append(messages, "SLACK_TOKEN is missing")
	}
	if c.OpsGenieToken == "" {
		messages = append(messages, "OPSGENIE_TOKEN is missing")
	}
	if c.PersonioClientId == "" {
		messages = append(messages, "PERSONIO_CLIENT_ID is missing")
	}
	if c.PersonioClientSecret == "" {
		messages = append(messages, "PERSONIO_CLIENT_SECRET is missing")
	}

	if len(messages) > 0 {
		return fmt.Errorf("the following required values are invalid or missing: %s", strings.Join(messages, ", "))
	}
	return nil
}
