package main

import (
	"context"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

type Slack struct {
	client        *slack.Client
	pagingEntries int
	userGroups    []slack.UserGroup
}

func NewSlack(token string, pagingEntries int) *Slack {
	s := Slack{
		client:        slack.New(token),
		pagingEntries: pagingEntries,
	}
	go s.getUserGroups()
	return &s
}

func (s *Slack) checkError(err error) {
	if rateLimitedError, ok := err.(*slack.RateLimitedError); ok {
		select {
		case <-time.After(rateLimitedError.RetryAfter):
			err = nil
		}
	}
	if err != nil {
		log.Println(err)
	}
}

func (s *Slack) getUserGroups() {
	var (
		users      chan []slack.UserGroup = make(chan []slack.UserGroup)
		pattern, _                        = regexp.Compile(SUPPORT_PATTERN)
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
			s.checkError(err)
		}
	}()
	s.userGroups = <-users
}

func (s *Slack) CreateUserGroup(name string, topics []string, members []string, ug chan slack.UserGroup) {
	var (
		description string          = "Support channel for requests relating to " + strings.Join(topics, ", ")
		group       slack.UserGroup = slack.UserGroup{
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
		if err == nil {
			break
		}
		s.checkError(err)
	}
	ug <- u
}

func (s *Slack) UpdateUserGroup(name, id string, topics []string, members []string) {
	var (
		description string = "Support channel for requests relating to " + strings.Join(topics, ", ")
		err         error
	)

	// These are deliberately separate loops.
	// merging them to a single will cause unnecessary nesting and potential hazards
	log.Println("Updating usergroup for", name)
	for {
		_, err = s.client.UpdateUserGroup(id, slack.UpdateUserGroupsOptionDescription(&description))
		if err == nil {
			break
		}
		s.checkError(err)
	}

	log.Println("Updating usergroup members for", name)
	for {
		var m string = strings.Join(members, ",")
		_, err = s.client.UpdateUserGroupMembers(id, m)
		if err == nil {
			break
		}
		s.checkError(err)
	}
}

func (s *Slack) CreateOrUpdateUserGroup(name string, topics []string, members []string) {
	var (
		usergroup chan slack.UserGroup = make(chan slack.UserGroup)
		existing  bool                 = false
		id        string               = ""
	)
	for _, item := range s.userGroups {
		if item.Name == name {
			existing = true
			id = item.ID
			break
		}
	}
	if existing {
		s.UpdateUserGroup(name, id, topics, members)
		return
	}
	go s.CreateUserGroup(name, topics, members, usergroup)
	if u, ok := <-usergroup; ok {
		s.userGroups = append(s.userGroups, u)
	}
}

func (s *Slack) GetUsersPaginated(matchDomain string, userchan *chan []Member) {
	log.Println("Retrieving users from slack")
	var (
		err         error
		membersChan chan Member = make(chan Member, 0)
		members     []Member    = make([]Member, 0)
		count       int         = 0
		done        chan bool   = make(chan bool, 0)
	)

	go func(membersChan *chan Member, count *int) {
		ctx := context.Background()
		p := s.client.GetUsersPaginated(slack.GetUsersOptionLimit(s.pagingEntries))
		for err == nil {
			p, err = p.Next(ctx)
			if err == nil {
				for _, user := range p.Users {
					if strings.HasSuffix(user.Profile.Email, matchDomain) {
						*count++
						go s.userGithubProfileViaChan(membersChan, Member{SlackID: user.ID, Email: user.Profile.Email})
					}
				}
			}
			s.checkError(err)
		}
		done <- true
	}(&membersChan, &count)

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

func (s *Slack) userGithubProfileViaChan(ch *chan Member, member Member) {
	member.GithubLogin = s.userGithubProfile(member.SlackID)
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
func (s *Slack) userGithubProfile(userID string) (github string) {
	var (
		options = slack.GetUserProfileParameters{
			UserID:        userID,
			IncludeLabels: false,
		}
		u          *slack.UserProfile
		err        error
		pattern, _ = regexp.Compile(GITHUB_URL_PATTERN)
	)

	// Attempts to handle retrieving from the API with rate limiting in place
	for {
		u, err = s.client.GetUserProfile(&options)
		if err == nil {
			break
		}
		s.checkError(err)
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

// Get topics from slack channels matching the team prefix.
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
	return
}

// Create slack handles for support
func (s *Slack) SlackHandles(teams []*Team) {
	var (
		supportTeams  = make(map[string][]string)
		teamTopics    = make(map[string][]string)
		supportTopics = make(map[string][]string)
	)
	for _, team := range teams {
		// TESTING
		// if team.Name != "team-honeybadger" {
		// 	continue
		// }
		var (
			supportName string   = "support-" + strings.Split(team.Name, "-")[1]
			members     []string = make([]string, 0)
		)
		teamTopics[supportName] = team.Topics
		for _, topic := range team.Topics {
			supportTopics = appendTopic(supportTopics, topic, supportName)
		}
		for _, m := range team.Members {
			var primary bool = (m.IsSolutionArchitect || m.IsAccountEngineer) && !m.Afk
			if (primary || m.Oncall) && m.SlackID != "" {
				members = append(members, m.SlackID)
			}
		}
		supportTeams[supportName] = members
	}

	for k, v := range supportTeams {
		s.CreateOrUpdateUserGroup(k, teamTopics[k], v)
	}

	for k, v := range supportTopics {
		var users []string = make([]string, 0)
		for _, handle := range v {
			users = append(users, supportTeams[handle]...)
		}
		// these only have a single topic - the second part of the support name
		s.CreateOrUpdateUserGroup(k, []string{strings.Split(k, "-")[1]}, users)
	}
}
