package planner

import (
	"encoding/json"
)

type Planner interface {
	Name() string
	IsRelevant(srcDir string) bool
	Plan(srcDir string) *BuildPlan
}

// TODO: decide if BuildPlan should continue to be a separate structure
// or whether it should be the same structure as devbox.Config.
type BuildPlan struct {
	Packages []string `cue:"[...string]" json:"packages"`
}

func (p *BuildPlan) String() string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

func Merge(plans ...*BuildPlan) *BuildPlan {
	plan := &BuildPlan{
		Packages: []string{},
	}
	for _, p := range plans {
		// TODO: de-duplicate
		plan.Packages = append(plan.Packages, p.Packages...)
	}
	return plan
}

var LANGUAGE_PLANNERS = []Planner{
	&GoPlanner{},
	&PythonPlanner{},
}

func Plan(srcDir string) *BuildPlan {
	result := &BuildPlan{
		Packages: []string{},
	}
	for _, planner := range LANGUAGE_PLANNERS {
		if planner.IsRelevant(srcDir) {
			plan := planner.Plan(srcDir)
			result = Merge(result, plan)
		}
	}
	return result
}
