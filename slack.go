package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

type Slack struct {
	client *slack.Client
}

func NewSlack(token string) *Slack {
	s := Slack{
		client: slack.New(token),
	}
	return &s
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
		} else if rateLimitedError, ok := err.(*slack.RateLimitedError); ok {
			select {
			case <-time.After(rateLimitedError.RetryAfter):
				err = nil
			}
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

// Get a list of Giantswarm users from Slack
//
// This is dreadful on performance for a number of reasons
// - Call to GetUsers returns all users in slack - we then have to
//   filter these by giantswarm email address to get just internal users
// - GetUsers does not return all profile information so for each internal
//   user we then need to call GetUserProfile in the method above.
//   This adds considerable overhead.
func (s *Slack) Users() (members []Member) {
	members = make([]Member, 0)
	users, _ := s.client.GetUsers()
	for _, user := range users {
		if !user.Deleted && strings.HasSuffix(user.Profile.Email, DOMAIN) {
			var github string = s.userGithubProfile(user.ID)
			var member = Member{
				SlackID:     user.ID,
				GithubLogin: github,
				Email:       user.Profile.Email,
			}

			members = append(members, member)
		}
	}
	return members
}

// Get topics from slack channels matching the team prefix.
func (s *Slack) Topics(match string) (topics map[string][]string) {
	topics = make(map[string][]string)
	var pattern, _ = regexp.Compile(match)
	slackChans := make([]slack.Channel, 0)
	initChans, initCur, err := s.client.GetConversations(
		&slack.GetConversationsParameters{
			ExcludeArchived: true,
			Limit:           1000,
			Types: []string{
				"public_channel",
				"private_channel",
			},
		},
	)
	if err != nil {
		fmt.Println(err)
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
				Limit:           1000,
				Types: []string{
					"public_channel",
					"private_channel",
				},
			},
		)
		if err != nil {
			fmt.Println(err)
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
	return
}

// Create slack handles for support
func (s *Slack) SlackHandles(teams []*Team) {
	var (
		supportTeams  = make(map[string][]string)
		supportTopics = make(map[string][]string)
	)
	for _, team := range teams {
		var (
			supportName string   = "support-" + strings.Split(team.Name, "-")[1]
			members     []string = make([]string, 0)
		)
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
	fmt.Println()
	for k, v := range supportTeams {
		fmt.Printf("%s: %v\n", k, v)
	}

	fmt.Println()
	for k, v := range supportTopics {
		var users []string = make([]string, 0)
		fmt.Printf("%s: %v (", k, v)
		for _, handle := range v {
			users = append(users, supportTeams[handle]...)
		}
		fmt.Printf("%v)  \n", users)
	}
}
