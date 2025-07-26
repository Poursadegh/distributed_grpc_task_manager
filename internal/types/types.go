package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
)

func (p Priority) String() string {
	switch p {
	case PriorityHigh:
		return "high"
	case PriorityMedium:
		return "medium"
	case PriorityLow:
		return "low"
	default:
		return "unknown"
	}
}

func ParsePriority(s string) Priority {
	switch s {
	case "high":
		return PriorityHigh
	case "medium":
		return PriorityMedium
	case "low":
		return PriorityLow
	default:
		return PriorityLow
	}
}

type Status int

const (
	StatusPending Status = iota
	StatusRunning
	StatusCompleted
	StatusFailed
	StatusTimeout
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

func ParseStatus(s string) Status {
	switch s {
	case "pending":
		return StatusPending
	case "running":
		return StatusRunning
	case "completed":
		return StatusCompleted
	case "failed":
		return StatusFailed
	case "timeout":
		return StatusTimeout
	default:
		return StatusPending
	}
}

type Task struct {
	ID          string          `json:"id"`
	Priority    Priority        `json:"priority"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   time.Time       `json:"created_at"`
	Status      Status          `json:"status"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	WorkerID    string          `json:"worker_id,omitempty"`
	Error       string          `json:"error,omitempty"`

	Dependencies []string       `json:"dependencies,omitempty"`
	Timeout      *time.Time     `json:"timeout,omitempty"`
	MaxDuration  *time.Duration `json:"max_duration,omitempty"`
}

func NewTask(priority Priority, payload json.RawMessage) *Task {
	return &Task{
		ID:        uuid.New().String(),
		Priority:  priority,
		Payload:   payload,
		CreatedAt: time.Now(),
		Status:    StatusPending,
	}
}

func NewTaskWithDependencies(priority Priority, payload json.RawMessage, dependencies []string) *Task {
	task := NewTask(priority, payload)
	task.Dependencies = dependencies
	return task
}

func NewTaskWithTimeout(priority Priority, payload json.RawMessage, maxDuration time.Duration) *Task {
	task := NewTask(priority, payload)
	task.MaxDuration = &maxDuration
	return task
}

func (t *Task) IsReadyToRun() bool {
	return t.Status == StatusPending && len(t.Dependencies) == 0
}

func (t *Task) HasTimedOut() bool {
	if t.MaxDuration == nil {
		return false
	}

	if t.StartedAt == nil {
		return false
	}

	return time.Since(*t.StartedAt) > *t.MaxDuration
}

type TaskRequest struct {
	Priority Priority        `json:"priority" binding:"required"`
	Payload  json.RawMessage `json:"payload" binding:"required"`
}

type TaskResponse struct {
	Task    *Task  `json:"task,omitempty"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type TasksResponse struct {
	Tasks   []*Task `json:"tasks"`
	Success bool    `json:"success"`
	Error   string  `json:"error,omitempty"`
}

type NodeInfo struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	IsLeader bool      `json:"is_leader"`
	LastSeen time.Time `json:"last_seen"`
	Status   string    `json:"status"`
}

type ClusterInfo struct {
	Nodes        []*NodeInfo `json:"nodes"`
	LeaderID     string      `json:"leader_id"`
	TotalTasks   int         `json:"total_tasks"`
	PendingTasks int         `json:"pending_tasks"`
	RunningTasks int         `json:"running_tasks"`
}
