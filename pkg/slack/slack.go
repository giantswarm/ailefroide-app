package slack

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	aile "github.com/giantswarm/ailefroide/pkg/ailefroide"
	ap "github.com/giantswarm/personio-go/v1"
	"github.com/slack-go/slack"
)

const (
	MAX_SLACK_USERGROUP_DESCRIPTION_LENGTH = 140
)

// Slack Min client struct for handling the slack api
type Slack struct {
	client        *slack.Client
	pagingEntries int
	userGroups    []slack.UserGroup
	teamSettings  map[string]aile.TeamSettings
}

// NewSlack Create a new Slack client object
func NewSlack(token, expression string, pagingEntries int, teamSettings map[string]aile.TeamSettings) *Slack {
	s := Slack{
		client:        slack.New(token),
		pagingEntries: pagingEntries,
		teamSettings:  teamSettings,
	}
	s.getUserGroups(expression)
	return &s
}

func (s *Slack) checkError(err error) error {
	if rateLimitedError, ok := err.(*slack.RateLimitedError); ok {
		<-time.After(rateLimitedError.RetryAfter)
		err = nil
	}
	if err != nil {
		if err.Error() != "pagination complete" {
			log.Printf("Slack client raised error '%s'", err.Error())
			debug.PrintStack()
		}
	}
	return err
}

func (s *Slack) getUserGroups(expression string) {
	var (
		users      = make(chan []slack.UserGroup)
		pattern, _ = regexp.Compile(expression)
	)
	defer close(users)
	go func() {
		for {
			ug, err := s.client.GetUserGroups()
			if err == nil {
				usergroups := make([]slack.UserGroup, 0)
				for _, item := range ug {
					if pattern.Match([]byte(item.Handle)) {
						usergroups = append(usergroups, item)
					}
				}
				users <- usergroups
				break
			}
			if err = s.checkError(err); err != nil {
				break
			}
		}
	}()
	s.userGroups = <-users
}

// CreateUserGroup Create a user group on the slack api
func (s *Slack) CreateUserGroup(name, description string, topics []string, members []string, ug chan slack.UserGroup) {
	var (
		group = slack.UserGroup{
			Name:        name,
			Handle:      name,
			Description: description,
			Users:       members,
		}
		u   slack.UserGroup
		err error
	)
	log.Println("Creating usergroup for", name)
	for err == nil {
		u, err = s.client.CreateUserGroup(group)
		if err != nil {
			if err.Error() == "description_too_long" {
				log.Println("description_too_long for ", name, description, len(description))
				break
			} else if err.Error() == "handle_already_exists" {
				u = s.updateUserGroup(name, u.ID, description, members)
			}
		} else if s.checkError(err) != nil {
			break
		}
	}
	ug <- u
}

// UpdateUserGroup Modify an existing user group
func (s *Slack) updateUserGroup(name, id, description string, members []string) slack.UserGroup {
	// Don't execute the function if the members list is empty.
	//
	// This is a workaround for a bug in the slack API where the members list
	// is not updated if the list is empty.
	//
	// Secondly, we do not want or desire to update the usergroup if there are
	// no members in it.
	if len(members) == 0 {
		return slack.UserGroup{}
	}

	var (
		u   slack.UserGroup
		err error
	)

	// These are deliberately separate loops.
	// merging them to a single will cause unnecessary nesting and potential hazards
	log.Println("Updating usergroup for", name)
	for {
		u, err = s.client.UpdateUserGroup(id, slack.UpdateUserGroupsOptionDescription(&description))
		if err != nil && err.Error() == "description_too_long" {
			log.Println("description_too_long for ", name, description, len(description))
		} else if err == nil || s.checkError(err) != nil {
			break
		}
	}

	log.Println("Updating usergroup members for", name)
	for {
		m := strings.Join(members, ",")
		_, err = s.client.UpdateUserGroupMembers(id, m)
		if err == nil || s.checkError(err) != nil {
			break
		}
	}

	return u
}

// CreateOrUpdateUserGroup Tries to work out of the group exists and updates it if so, else creates it
func (s *Slack) CreateOrUpdateUserGroup(name string, topics []string, members []string, done *chan bool) {
	var (
		description = "Support channel for requests relating to " + strings.Join(topics, ", ")
		usergroup   = make(chan slack.UserGroup)
		existing    = false
		id          = ""
	)

	pattern := regexp.MustCompile(`[^a-zA-Z0-9,-. ]+`)
	description = pattern.ReplaceAllString(description, "")

	if len(description) > MAX_SLACK_USERGROUP_DESCRIPTION_LENGTH {
		log.Printf("Truncating description %q to %d\n", description, MAX_SLACK_USERGROUP_DESCRIPTION_LENGTH)
		description = description[:MAX_SLACK_USERGROUP_DESCRIPTION_LENGTH-3] + "..."
	}

	for _, item := range s.userGroups {
		if item.Handle == name {
			existing = true
			id = item.ID
			break
		}
	}

	if id == "" {
		log.Println("unable to discover ID for usergroup", name)
	}
	if existing {
		_ = s.updateUserGroup(name, id, description, members)
		*done <- true
		return
	}
	go s.CreateUserGroup(name, description, topics, members, usergroup)
	if u, ok := <-usergroup; ok {
		s.userGroups = append(s.userGroups, u)
	}
	*done <- true
}

// GetPersonioUsersPaginated Gets slack users from the given list of personio users
func (s *Slack) GetPersonioUsersPaginated(matchDomain string, people []*ap.Employee, userchan *chan []aile.Member, github string) {
	log.Println("Retrieving personio users from slack")
	var (
		err         error
		membersChan = make(chan aile.Member)
		members     = make([]aile.Member, 0)
		count       = 0
		done        = make(chan bool)
	)

	go func(membersChan *chan aile.Member, count *int, people []*ap.Employee) {
		ctx := context.Background()
		p := s.client.GetUsersPaginated(slack.GetUsersOptionLimit(s.pagingEntries))
		for err == nil {
			p, err = p.Next(ctx)
			if s.checkError(err) != nil {
				break
			}

			for _, user := range p.Users {
				go func(user slack.User) {
					for _, person := range people {
						email := person.GetStringAttribute("email")
						if strings.EqualFold(user.Profile.Email, *email) {
							*count++
							*membersChan <- aile.Member{
								Email:       *email,
								SlackID:     user.ID,
								GithubLogin: strings.ToLower(*person.GetStringAttribute(github)),
							}
						}
					}
				}(user)
			}
		}
		done <- true
	}(&membersChan, &count, people)

	<-done
	i := 0
	for i < count {
		user := <-membersChan
		members = append(members, user)
		i++
	}

	log.Println("Done retrieving slack users")
	*userchan <- members
}

// Topics Get topics from slack channels matching the team prefix.
func (s *Slack) Topics(match string, topchan *chan map[string][]string) {
	log.Println("Retrieving topics from slack")
	topics := make(map[string][]string)
	pattern, _ := regexp.Compile(match)
	slackChans := make([]slack.Channel, 0)
	initChans, initCur, err := s.client.GetConversations(
		&slack.GetConversationsParameters{
			ExcludeArchived: true,
			Limit:           s.pagingEntries,
			Types: []string{
				"public_channel",
				"private_channel",
			},
		},
	)
	if err != nil {
		log.Println(err)
		return
	}

	slackChans = append(slackChans, initChans...)

	// Paginate over additional channels
	nextCur := initCur
	for nextCur != "" {
		channels, cursor, err := s.client.GetConversations(
			&slack.GetConversationsParameters{
				Cursor:          nextCur,
				ExcludeArchived: true,
				Limit:           s.pagingEntries,
				Types: []string{
					"public_channel",
					"private_channel",
				},
			},
		)
		if err != nil {
			log.Println(err)
			return
		}

		slackChans = append(slackChans, channels...)
		nextCur = cursor
	}
	for _, channel := range slackChans {
		if pattern.Match([]byte(channel.Name)) {
			log.Printf("Checking channel '%q' topic '%q'\n", channel.Name, channel.Topic.Value)
			topicList := strings.Split(channel.Topic.Value, ",")
			topics[channel.Name] = make([]string, 0)
			for _, item := range topicList {
				topics[channel.Name] = append(topics[channel.Name], strings.TrimSpace(item))
			}
		}
	}
	*topchan <- topics
	log.Println("Done retrieving slack topics")
}

// SlackHandles Create slack handles for support
func (s *Slack) SlackHandles(teams []*aile.Team, debug bool, debugTeam string) {
	var (
		supportTeams    = make(map[string][]string)
		teamTopics      = make(map[string][]string)
		supportTopics   = make(map[string][]string)
		defaultSettings = aile.TeamSettings{
			SkipRotation:             true,
			IncludeOnCallEngineer:    true,
			IncludeProductOwner:      true,
			IncludeSRE:               false,
			IncludePlatformArchitect: false,
		}
	)

	for _, team := range teams {
		f := false
		for k := range s.teamSettings {
			if k == team.Name {
				f = true
				break
			}
		}

		if !f {
			s.teamSettings[team.Name] = defaultSettings
		}

		var (
			supportName  = "support-" + strings.Split(team.Name, "-")[1]
			members      = make([]string, 0)
			teamSettings = s.teamSettings[team.Name]
		)

		if teamSettings.SkipRotation || (debugTeam != "" && team.Name != debugTeam) {
			continue
		}

		teamTopics[supportName] = team.Topics

		for _, topic := range team.Topics {
			supportTopics = aile.AppendTopic(supportTopics, topic, supportName)
		}

		for _, m := range team.Members {
			var (
				includeProductOwner     = (m.IsProductOwner && teamSettings.IncludeProductOwner)
				includePlaformArchitect = (m.IsPlatformArchitect && teamSettings.IncludePlatformArchitect)
				includeSRE              = (m.IsSiteReliabilityEngineer && teamSettings.IncludeSRE)
				includeOncall           = (m.Oncall && teamSettings.IncludeOnCallEngineer)
				primary                 = (m.IsSolutionArchitect || m.IsAccountEngineer || includeProductOwner)
				secondary               = (includeOncall || includePlaformArchitect || includeSRE)
			)

			if m.Email == "" {
				log.Printf("Skipping user with no email address %+v", m)
			}
			if m.Oncall && m.Afk {
				m.Afk = false
			}

			message := fmt.Sprintf("%s is NOT", m.Email)
			if (primary || secondary || m.IncludeWhenNotAFK) && m.SlackID != "" && !m.Afk {
				message = fmt.Sprintf("%s is", m.Email)
				if debug {
					members = append(members, fmt.Sprintf("%s (%s)", m.Email, m.SlackID))
				} else {
					members = append(members, m.SlackID)
				}
			}
			message += fmt.Sprintf(" because primary=%t secondary=%t afk=%t slackid=%s", primary, secondary, m.Afk, m.SlackID)
			log.Println(message)
		}

		supportTeams[supportName] = members
	}

	var (
		teamsDone  = make(chan bool)
		topicsDone = make(chan bool)
		te, to     = 0, 0
	)

	if debug {
		s.debug(supportTeams, supportTopics)
		return
	}

	for k, v := range supportTeams {
		go s.CreateOrUpdateUserGroup(k, teamTopics[k], v, &teamsDone)
	}

	for k, v := range supportTopics {
		users := make([]string, 0)
		for _, handle := range v {
			users = append(users, supportTeams[handle]...)
		}
		// these only have a single topic - the second part of the support name
		go s.CreateOrUpdateUserGroup(k, []string{strings.Split(k, "-")[1]}, users, &topicsDone)
	}

	for {
		select {
		case <-teamsDone:
			te++
		case <-topicsDone:
			to++
		}

		if te >= len(supportTeams) && to >= len(supportTopics) {
			break
		}
	}
}

func (s *Slack) debug(supportTeams, supportTopics map[string][]string) {
	for k, v := range supportTeams {
		fmt.Printf("%s: %v\n", k, v)
	}

	for k, v := range supportTopics {
		users := make([]string, 0)
		fmt.Printf("%s: %v (", k, v)
		for _, handle := range v {
			users = append(users, supportTeams[handle]...)
		}
		fmt.Printf("%v)  \n", users)
	}
}
