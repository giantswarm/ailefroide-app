package ailefroide

// Team member information
//
// Contains combined information from all APIs about a particular engineer
type Member struct {
	GithubLogin               string
	Email                     string
	SlackID                   string
	IsSolutionArchitect       bool
	IsAccountEngineer         bool
	IsProductOwner            bool
	IsPlatformArchitect       bool
	IsSiteReliabilityEngineer bool
	Afk                       bool
	Oncall                    bool
	IncludeWhenNotAFK         bool
}

// Team Details of a individual team
//
// This is the information which will be used to map to the support handle
type Team struct {
	Name    string
	Topics  []string
	Members []*Member
}

// HasSolutionArchitect Is there a solution architect in this team
func (t *Team) HasSolutionArchitect() bool {
	for _, m := range t.Members {
		if m.IsSolutionArchitect {
			return true
		}
	}
	return false
}

// HasAccountEngineer Is there an account engineer in this team
func (t *Team) HasAccountEngineer() bool {
	for _, m := range t.Members {
		if m.IsAccountEngineer {
			return true
		}
	}
	return false
}

// HasProductOwner Is there a product owner in this team
func (t *Team) HasProductOwner() bool {
	for _, m := range t.Members {
		if m.IsProductOwner {
			return true
		}
	}
	return false
}

// IsEngineeringTeam Helper function to check if this is an engineering team
func (t *Team) IsEngineeringTeam() bool {
	return t.HasSolutionArchitect()
}

// IsAccountEngineering Helper function to test if this is the account engineering team
func (t *Team) IsAccountEngineering() bool {
	for _, m := range t.Members {
		if !m.IsAccountEngineer {
			return false
		}
	}
	return true
}

// InRotation Should this team get a support handle
//
// A team qualifies by its make-up: it has a solution architect, or it is the
// account engineering team. A team without either can opt in with
// rotateWithoutSolutionArchitect; otherwise it drops out, and its handles keep
// their last members, since nothing updates them any more.
func (t *Team) InRotation(settings map[string]TeamSettings) bool {
	if s, ok := settings[t.Name]; ok && !s.SkipRotation && s.RotateWithoutSolutionArchitect {
		return true
	}
	return t.IsEngineeringTeam() || t.IsAccountEngineering()
}
