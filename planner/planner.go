package planner

type Planner interface {
	Name() string
	IsRelevant(srcDir string) bool
	Plan(srcDir string) *BuildPlan
}

var PLANNERS = []Planner{
	&GoPlanner{},
	&PythonPlanner{},
}

func Plan(srcDir string) *BuildPlan {
	result := &BuildPlan{
		Packages: []string{},
	}
	for _, planner := range PLANNERS {
		if planner.IsRelevant(srcDir) {
			plan := planner.Plan(srcDir)
			result = MergePlans(result, plan)
		}
	}
	return result
}
