package planner

type PythonPlanner struct{}

// PythonPlanner implements interface Planner (compile-time check)
var _ Planner = (*PythonPlanner)(nil)

func (g *PythonPlanner) Name() string {
	return "PythonPlanner"
}

func (g *PythonPlanner) IsRelevant(srcDir string) bool {
	return false
}

func (g *PythonPlanner) Plan(srcDir string) *BuildPlan {
	return &BuildPlan{}
}
