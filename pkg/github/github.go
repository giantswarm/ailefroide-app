package github

import (
	"context"
	"log"
	"regexp"

	aile "github.com/giantswarm/ailefroide/pkg/ailefroide"
	"github.com/google/go-github/v47/github"
	"golang.org/x/oauth2"
)

type Github struct {
	client             *github.Client
	organisation       string
	se                 string
	ae                 string
	solutionArchitects []*aile.Member
	accountEngineers   []*aile.Member
	productOwners      []*aile.Member
}

func NewGithub(cfg *aile.Config) *Github {
	log.Println("Setting up Github")
	g := Github{
		organisation: cfg.Organisation,
		se:           cfg.SolutionArchitects,
		ae:           cfg.AccountEngineers,
	}

	var (
		tokenSource = oauth2.StaticTokenSource(
			&oauth2.Token{
				AccessToken: cfg.GithubToken,
			},
		)
		ctx         = context.Background()
		oauthClient = oauth2.NewClient(ctx, tokenSource)
	)
	g.client = github.NewClient(oauthClient)

	g.solutionArchitects = make([]*aile.Member, 0)
	g.accountEngineers = make([]*aile.Member, 0)
	g.productOwners = make([]*aile.Member, 0)

	log.Println("Retrieving principle teams from github")
	if cfg.SolutionArchitects != "" {
		g.solutionArchitects = g.getMembers(g.organisation, cfg.SolutionArchitects)
	}
	if cfg.AccountEngineers != "" {
		g.accountEngineers = g.getMembers(g.organisation, cfg.AccountEngineers)
	}
	if cfg.ProductOwners != "" {
		g.productOwners = g.getMembers(g.organisation, cfg.ProductOwners)
	}
	log.Println("Done setting up github")

	return &g
}

func (g *Github) Teams(org, match string, teamschan *chan []*aile.Team) {
	log.Println("Retrieving teams from github")
	var (
		ctx        = context.Background()
		opts       = &github.ListOptions{}
		pattern, _ = regexp.Compile(match)
	)

	var (
		teams                   = make([]*aile.Team)
		teamchan chan aile.Team = make(chan aile.Team)
		count, i int            = 0, 0
	)
	defer close(teamchan)

	for {
		t, r, e := g.client.Teams.ListTeams(ctx, org, opts)
		if e != nil {
			log.Println(e)
			return
		}
		for _, item := range t {
			var name string = item.GetSlug()
			if pattern.Match([]byte(name)) {
				count++
				go g.getTeamViaChannel(g.organisation, name, &teamchan)
			}
		}

		if r.NextPage == 0 {
			break
		}
		opts.Page = r.NextPage
	}

	for i < count {
		var team aile.Team = <-teamchan
		teams = append(teams, &team)
		i++
	}
	log.Println("Done retrieving github teams")
	*teamschan <- teams
}

func (g *Github) getTeamViaChannel(org, team string, teamchan *chan aile.Team) {
	var members []*aile.Member = g.getMembers(org, team)
	*teamchan <- aile.Team{
		Name:    team,
		Members: members,
	}
}

func (g *Github) getMembers(org, team string) (members []*aile.Member) {
	var (
		ctx  = context.Background()
		opts = &github.TeamListTeamMembersOptions{}
	)

	members = make([]*aile.Member, 0)

	for {
		u, r, e := g.client.Teams.ListTeamMembersBySlug(ctx, org, team, opts)
		if e != nil {
			log.Println(e)
			return
		}

		for _, item := range u {
			var login string = item.GetLogin()
			var member aile.Member = aile.Member{
				GithubLogin: login,
			}
			if team != g.ae && team != g.se {
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

func (g *Github) containsLogin(login string, team []*aile.Member) bool {
	for _, member := range team {
		if login == member.GithubLogin {
			return true
		}
	}
	return false
}
