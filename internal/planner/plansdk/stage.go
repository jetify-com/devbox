package plansdk

type Stage struct { // TODO: remove it?
	Command string `cue:"string" json:"command"`
	// InputFiles is internal for planners only.
	InputFiles []string `cue:"[...string]" json:"input_files,omitempty"`
	// Warning is internal for planners only.
	// If a stage has Warning, we will print it if
	// a command override is not present in devbox.json
	Warning error `json:"warning,omitempty"`
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
