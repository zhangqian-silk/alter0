package domain

const (
	MemoryContextMetadataKey     = "alter0.memory_context"
	MemoryContextProtocolVersion = "alter0.memory-context/v1"
)

type MemoryContext struct {
	Protocol string            `json:"protocol"`
	Files    []MemoryFileSpec  `json:"files"`
	Recall   []MemoryRecallHit `json:"recall,omitempty"`
}

type MemoryFileSpec struct {
	ID        string `json:"id"`
	Selection string `json:"selection"`
	Title     string `json:"title"`
	Path      string `json:"path"`
	Exists    bool   `json:"exists"`
	Writable  bool   `json:"writable"`
	UpdatedAt string `json:"updated_at,omitempty"`
	Content   string `json:"content,omitempty"`
}

type MemoryRecallHit struct {
	MemoryID string `json:"memory_id"`
	Title    string `json:"title"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Snippet  string `json:"snippet"`
}
