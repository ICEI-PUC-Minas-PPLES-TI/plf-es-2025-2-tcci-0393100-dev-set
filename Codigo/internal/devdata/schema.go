package devdata

// SeedData represents the structure of the seed data JSON file
type SeedData struct {
	Metadata SeedMetadata `json:"metadata"`
	Tasks    []SeedTask   `json:"tasks"`
}

// SeedMetadata contains information about the seed data
type SeedMetadata struct {
	Version     string   `json:"version"`
	Source      string   `json:"source"`
	Description string   `json:"description"`
	Categories  []string `json:"categories"`
	TotalTasks  int      `json:"total_tasks"`
}

// SeedTask represents a single task in the seed data
type SeedTask struct {
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Labels         []string               `json:"labels"`
	Category       string                 `json:"category"`
	ActualHours    float64                `json:"actual_hours"`
	EstimatedHours float64                `json:"estimated_hours,omitempty"`
	StoryPoints    int                    `json:"story_points"`
	Size           string                 `json:"size"`
	Complexity     string                 `json:"complexity,omitempty"`
	State          string                 `json:"state"` // "open" or "closed"
	Priority       string                 `json:"priority,omitempty"`
	CustomFields   map[string]interface{} `json:"custom_fields,omitempty"`
}
