package planner

type EmptyPlanner struct{}

// EmptyPlanner implements interface Planner (compile-time check)
var _ Planner = (*EmptyPlanner)(nil)

func (g *EmptyPlanner) Name() string {
	return "EmptyPlanner"
}

func (g *EmptyPlanner) IsRelevant(srcDir string) bool {
	return false
}

func (g *EmptyPlanner) Plan(srcDir string) *BuildPlan {
	return &BuildPlan{}
}
