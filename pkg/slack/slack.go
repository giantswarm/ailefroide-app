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
	ap "github.com/giantswarm/ailefroide/pkg/personio"
	"github.com/slack-go/slack"
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
	go s.getUserGroups(expression)
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
		users      chan []slack.UserGroup = make(chan []slack.UserGroup)
		pattern, _                        = regexp.Compile(expression)
	)
	defer close(users)
	go func() {
		var err error
		for err == nil {
			ug, err := s.client.GetUserGroups()
			if err == nil {
				var usergroups []slack.UserGroup = make([]slack.UserGroup, 0)
				for _, item := range ug {
					if pattern.Match([]byte(item.Name)) {
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
		group slack.UserGroup = slack.UserGroup{
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
		if err == nil || s.checkError(err) != nil {
			break
		}
	}
	ug <- u
}

// UpdateUserGroup Modify an existing user group
func (s *Slack) UpdateUserGroup(name, id, description string, topics []string, members []string) {
	var err error

	// These are deliberately separate loops.
	// merging them to a single will cause unnecessary nesting and potential hazards
	log.Println("Updating usergroup for", name)
	for {
		_, err = s.client.UpdateUserGroup(id, slack.UpdateUserGroupsOptionDescription(&description))
		if err == nil || s.checkError(err) != nil {
			break
		}
	}

	log.Println("Updating usergroup members for", name)
	for {
		var m string = strings.Join(members, ",")
		_, err = s.client.UpdateUserGroupMembers(id, m)
		if err == nil || s.checkError(err) != nil {
			break
		}
	}
}

// CreateOrUpdateUserGroup Tries to work out of the group exists and updates it if so, else creates it
func (s *Slack) CreateOrUpdateUserGroup(name string, topics []string, members []string, done *chan bool) {
	var (
		description string               = "Support channel for requests relating to " + strings.Join(topics, ", ")
		usergroup   chan slack.UserGroup = make(chan slack.UserGroup)
		existing    bool                 = false
		id          string               = ""
	)
	for _, item := range s.userGroups {
		if item.Name == name {
			existing = true
			id = item.ID
			break
		}
	}
	if len(members) != 0 { // updating with an empty members list causes an infinite loop.
		if existing {
			s.UpdateUserGroup(name, id, description, topics, members)
			*done <- true
			return
		}
		go s.CreateUserGroup(name, description, topics, members, usergroup)
		if u, ok := <-usergroup; ok {
			s.userGroups = append(s.userGroups, u)
		}
	}
	*done <- true
}

// GetUsersPaginated Gets the list of users from Slack
func (s *Slack) GetUsersPaginated(matchDomain, expression string, userchan *chan []aile.Member) {
	log.Println("Retrieving users from slack")
	var (
		err         error
		count       int              = 0
		membersChan chan aile.Member = make(chan aile.Member)
		members     []aile.Member    = make([]aile.Member, 0)
		done        chan bool        = make(chan bool)
	)

	go func(membersChan *chan aile.Member, count *int, expression string) {
		ctx := context.Background()
		p := s.client.GetUsersPaginated(slack.GetUsersOptionLimit(s.pagingEntries))
		for err == nil {
			p, err = p.Next(ctx)
			if err == nil {
				for _, user := range p.Users {
					if strings.HasSuffix(user.Profile.Email, matchDomain) {
						*count++
						go s.userGithubProfileViaChan(membersChan, aile.Member{
							SlackID: user.ID,
							Email:   user.Profile.Email,
						}, expression)
					}
				}
			}
			if err = s.checkError(err); err != nil {
				break
			}
		}
		done <- true
	}(&membersChan, &count, expression)

	var i int = 0

	<-done
	for i < count {
		select {
		case user := <-membersChan:
			members = append(members, user)
			i++
		case <-done:
			break
		}
	}

	log.Println("Done retrieving slack users")
	*userchan <- members
}

// GetPersonioUsersPaginated Gets slack users from the given list of personio users
func (s *Slack) GetPersonioUsersPaginated(matchDomain string, people []ap.Employee, userchan *chan []aile.Member) {
	log.Println("Retrieving personio users from slack")
	var (
		err         error
		membersChan chan aile.Member = make(chan aile.Member)
		members     []aile.Member    = make([]aile.Member, 0)
		count       int              = 0
		done        chan bool        = make(chan bool)
	)

	go func(membersChan *chan aile.Member, count *int, people []ap.Employee) {
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
						if strings.ToLower(user.Profile.Email) == person.Email {
							*count++
							*membersChan <- aile.Member{
								Email:       person.Email,
								SlackID:     user.ID,
								GithubLogin: strings.ToLower(person.Github),
							}
						}
					}
				}(user)
			}
		}
		done <- true
	}(&membersChan, &count, people)

	<-done
	var i int = 0
	for i < count {
		select {
		case user := <-membersChan:
			members = append(members, user)
			i++
		case <-done:
			break
		}
	}

	log.Println("Done retrieving slack users")
	*userchan <- members

}

func (s *Slack) userGithubProfileViaChan(ch *chan aile.Member, member aile.Member, expression string) {
	member.GithubLogin = s.userGithubProfile(member.SlackID, expression)
	*ch <- member
}

// Meh.
// This method is garbage but needs to exist because things.
//
// Basically internal users may not have their GS email as their primary in Github.
// This makes it hard to match github users to slack users.
// To get around this, we retrieve the users profile, then try and match their github
// handle to the github URL field on the users profile.
// Of course this relies on the user actually having their profile set... which they
// may well not have.
//
// Automation only works when users want to be automated.
//
func (s *Slack) userGithubProfile(userID, expression string) (github string) {
	var (
		options = slack.GetUserProfileParameters{
			UserID:        userID,
			IncludeLabels: false,
		}
		u          *slack.UserProfile
		err        error
		pattern, _ = regexp.Compile(expression)
	)

	// Attempts to handle retrieving from the API with rate limiting in place
	for {
		u, err = s.client.GetUserProfile(&options)
		if err == nil || s.checkError(err) != nil {
			break
		}
	}

	if u != nil && u.Fields.Len() > 0 {
		for _, v := range u.Fields.ToMap() {
			match := pattern.FindStringSubmatch(v.Value)
			if len(match) != 0 {
				github = match[1]
			}
		}
	}

	return
}

// Topics Get topics from slack channels matching the team prefix.
func (s *Slack) Topics(match string, topchan *chan map[string][]string) {
	log.Println("Retrieving topics from slack")
	var topics = make(map[string][]string)
	var pattern, _ = regexp.Compile(match)
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
			var topicList []string = strings.Split(channel.Topic.Value, ",")
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
		supportTeams                      = make(map[string][]string)
		teamTopics                        = make(map[string][]string)
		supportTopics                     = make(map[string][]string)
		defaultSettings aile.TeamSettings = aile.TeamSettings{
			SkipRotation:             true,
			IncludeOnCallEngineer:    true,
			IncludeProductOwner:      true,
			IncludeSRE:               false,
			IncludePlatformArchitect: false,
		}
	)

	for _, team := range teams {
		var f bool = false
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
			supportName  string            = "support-" + strings.Split(team.Name, "-")[1]
			members      []string          = make([]string, 0)
			teamSettings aile.TeamSettings = s.teamSettings[team.Name]
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
				includeProductOwner     bool = (m.IsProductOwner && teamSettings.IncludeProductOwner)
				includePlaformArchitect bool = (m.IsPlatformArchitect && teamSettings.IncludePlatformArchitect)
				includeSRE              bool = (m.IsSiteReliabilityEngineer && teamSettings.IncludeSRE)
				includeOncall           bool = (m.Oncall && teamSettings.IncludeOnCallEngineer)
				primary                 bool = (m.IsSolutionArchitect || m.IsAccountEngineer || includeProductOwner)
				secondary               bool = (includeOncall || includePlaformArchitect || includeSRE)
			)
			if (primary || secondary) && m.SlackID != "" && !m.Afk {
				if debug {
					members = append(members, m.Email)
				} else {
					members = append(members, m.SlackID)
				}
			}
		}

		supportTeams[supportName] = members
	}

	var (
		teamsDone  chan bool = make(chan bool)
		topicsDone chan bool = make(chan bool)
		te, to     int       = 0, 0
	)

	if debug {
		s.debug(supportTeams, supportTopics)
		return
	}

	for k, v := range supportTeams {
		go s.CreateOrUpdateUserGroup(k, teamTopics[k], v, &teamsDone)
	}

	for k, v := range supportTopics {
		var users []string = make([]string, 0)
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
		var users []string = make([]string, 0)
		fmt.Printf("%s: %v (", k, v)
		for _, handle := range v {
			users = append(users, supportTeams[handle]...)
		}
		fmt.Printf("%v)  \n", users)
	}
}
