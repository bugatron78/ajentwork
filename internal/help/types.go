package help

type ArgDoc struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type OptionDoc struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ExampleDoc struct {
	Label   string `json:"label"`
	Command string `json:"command"`
}

type Doc struct {
	Name           string      `json:"name"`
	Summary        string      `json:"summary"`
	Purpose        string      `json:"purpose"`
	Usage          string      `json:"usage"`
	Arguments      []ArgDoc    `json:"arguments,omitempty"`
	Options        []OptionDoc `json:"options,omitempty"`
	Examples       []ExampleDoc `json:"examples,omitempty"`
	Related        []string    `json:"related,omitempty"`
	Safety         []string    `json:"safety,omitempty"`
	WorkflowTags   []string    `json:"workflow_tags,omitempty"`
	SearchKeywords []string    `json:"search_keywords,omitempty"`
}

type CommandSummary struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Group   string `json:"group"`
}

type WorkflowDoc struct {
	Name  string   `json:"name"`
	Topic string   `json:"topic"`
	Steps []string `json:"steps"`
}

type ExampleSet struct {
	Topic    string       `json:"topic"`
	Examples []ExampleDoc `json:"examples"`
}

type GlossaryEntry struct {
	Term       string `json:"term"`
	Definition string `json:"definition"`
}

type SearchResult struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
}
