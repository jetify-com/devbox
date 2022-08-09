package planner

import "encoding/json"

// TODO: decide if BuildPlan should continue to be a separate structure
// or whether it should be the same structure as devbox.Config.
type BuildPlan struct {
	Packages       []string `cue:"[...string]" json:"packages"`
	InstallCommand string   `cue:"string" json:"install_command,omitempty"`
	BuildCommand   string   `cue:"string" json:"build_command,omitempty"`
	StartCommand   string   `cue:"string" json:"start_command,omitempty"`
}

func (p *BuildPlan) String() string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

func MergePlans(plans ...*BuildPlan) *BuildPlan {
	plan := &BuildPlan{
		Packages: []string{},
	}
	for _, p := range plans {
		// TODO: de-duplicate
		plan.Packages = append(plan.Packages, p.Packages...)
	}
	return plan
}
