package plansdk

type Stage struct {
	Command string `cue:"string" json:"command"`
	// InputFiles is internal for planners only.
	InputFiles []string `cue:"[...string]" json:"input_files,omitempty"`
}

func (s *Stage) GetCommand() string {
	if s == nil {
		return ""
	}
	return s.Command
}

func (s *Stage) GetInputFiles() []string {
	if s == nil {
		return []string{}
	}
	return s.InputFiles
}

func AllFiles() []string {
	return []string{"."}
}
