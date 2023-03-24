package github

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	aile "github.com/giantswarm/ailefroide/pkg/ailefroide"
	"github.com/google/go-github/v47/github"
)

type Github struct {
	client             *github.Client
	organisation       string
	se                 string
	ae                 string
	po                 string
	solutionArchitects []*aile.Member
	accountEngineers   []*aile.Member
	productOwners      []*aile.Member
	cfg                *aile.Config
}

func NewGithub(cfg *aile.Config) *Github {
	log.Println("Setting up Github")
	g := Github{
		organisation: cfg.Organisation,
		se:           cfg.SolutionArchitects,
		ae:           cfg.AccountEngineers,
		po:           cfg.ProductOwners,
		cfg:          cfg,
	}

	itr, _ := ghinstallation.NewKeyFromFile(http.DefaultTransport, cfg.Gh.AppId, cfg.Gh.InstallationId, cfg.Gh.PrivateKey)
	g.client = github.NewClient(&http.Client{Transport: itr})

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
		teams                   = make([]*aile.Team, 0)
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
			if team != g.ae && team != g.se && team != g.po {
				member.IsAccountEngineer = g.containsLogin(login, g.accountEngineers)
				member.IsProductOwner = g.containsLogin(login, g.productOwners)
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

	// Add in any additional users from config
	var exists bool = false
	for k := range g.cfg.Teams {
		if k == team {
			exists = true
			break
		}
	}

	if exists {
		for _, m := range g.cfg.Teams[team].ExtraCover {
			m = strings.Trim(m, "@")
			m = strings.ToLower(m)
			var member aile.Member = aile.Member{}

			member.GithubLogin = m
			if strings.Contains(m, "@") {
				member.Email = m
				member.GithubLogin = ""
			}

			member.IsAccountEngineer = g.containsLogin(m, g.accountEngineers)
			member.IsProductOwner = g.containsLogin(m, g.productOwners)
			// You can be an account engineer, or a solution architect but not both.
			member.IsSolutionArchitect = g.containsLogin(m, g.solutionArchitects) && !member.IsAccountEngineer
			members = append(members, &member)
		}
	}

	return
}

func (g *Github) IsAccountEngineer(login string) bool {
	return g.containsLogin(login, g.accountEngineers)
}

func (g *Github) IsProductOwner(login string) bool {
	return g.containsLogin(login, g.productOwners)
}

func (g *Github) IsSolutionArchitect(login string) bool {
	return g.containsLogin(login, g.solutionArchitects)
}

func (g *Github) containsLogin(login string, team []*aile.Member) bool {
	for _, member := range team {
		if strings.EqualFold(login, member.GithubLogin) {
			return true
		}
	}
	return false
}
