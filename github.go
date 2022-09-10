package main

import (
	"context"
	"log"
	"regexp"

	"github.com/google/go-github/v47/github"
	"golang.org/x/oauth2"
)

type Github struct {
	client             *github.Client
	solutionArchitects []*Member
	accountEngineers   []*Member
}

func NewGithub(token string) *Github {
	g := Github{}

	var (
		tokenSource = oauth2.StaticTokenSource(
			&oauth2.Token{
				AccessToken: token,
			},
		)
		ctx         = context.Background()
		oauthClient = oauth2.NewClient(ctx, tokenSource)
	)
	g.client = github.NewClient(oauthClient)
	g.solutionArchitects = g.getMembers(ORGANISATION, SOLUTION_ARCHITECTS)
	g.accountEngineers = g.getMembers(ORGANISATION, ACCOUNT_ENGINEERS)
	return &g
}

func (g *Github) Teams(org, match string) (teams []*Team) {
	var (
		ctx        = context.Background()
		opts       = &github.ListOptions{}
		pattern, _ = regexp.Compile(match)
	)

	teams = make([]*Team, 0)

	for {
		t, r, e := g.client.Teams.ListTeams(ctx, org, opts)
		if e != nil {
			log.Println(e)
			return
		}
		for _, item := range t {
			var name string = item.GetSlug()
			if pattern.Match([]byte(name)) {
				var team Team = Team{
					Name:    name,
					Members: g.getMembers(ORGANISATION, name),
				}
				teams = append(teams, &team)
			}
		}

		if r.NextPage == 0 {
			break
		}
		opts.Page = r.NextPage
	}
	return
}

func (g *Github) getMembers(org, team string) (members []*Member) {
	var (
		ctx  = context.Background()
		opts = &github.TeamListTeamMembersOptions{}
	)

	members = make([]*Member, 0)

	for {
		u, r, e := g.client.Teams.ListTeamMembersBySlug(ctx, org, team, opts)
		if e != nil {
			log.Println(e)
			return
		}

		for _, item := range u {
			var login string = item.GetLogin()
			var member Member = Member{
				GithubLogin: login,
			}
			if team != ACCOUNT_ENGINEERS && team != SOLUTION_ARCHITECTS {
				member.IsAccountEngineer = g.containsLogin(login, g.accountEngineers)
				// You can be an account engineer, or a solution architect but not both.
				member.IsSolutionArchitect = g.containsLogin(login, g.solutionArchitects) && !member.IsAccountEngineer
			}

			members = append(members, &member)

		}

		if r.NextPage == 0 {
			break
		}

		opts.Page = r.NextPage
	}
	return
}

func (g *Github) containsLogin(login string, team []*Member) bool {
	for _, member := range team {
		if login == member.GithubLogin {
			return true
		}
	}
	return false
}
