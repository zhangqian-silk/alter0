package task

type Task struct {
	ID            string
	UserID        string
	Title         string
	Status        string
	CreatedAt     int64
	UpdatedAt     int64
	ClosedAt      int64
	LastChannelID string
}

type TaskMessage struct {
	ID        string
	TaskID    string
	UserID    string
	ChannelID string
	Role      string
	Content   string
	CreatedAt int64
	Meta      map[string]interface{}
}

type TaskMemory struct {
	TaskID    string
	Summary   string
	UpdatedAt int64
}

type TaskMemorySnapshot struct {
	UserID    string
	TaskID    string
	Summary   string
	UpdatedAt int64
}

type RouteDecision struct {
	Decision   string  `json:"decision"`
	TaskID     string  `json:"task_id,omitempty"`
	Title      string  `json:"title,omitempty"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason,omitempty"`
}

type CloseDecision struct {
	Close      bool    `json:"close"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason,omitempty"`
}
