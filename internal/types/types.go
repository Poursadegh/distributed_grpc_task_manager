package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Priority represents the priority level of a task
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
)

// String returns the string representation of the priority
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

// ParsePriority parses a string into a Priority
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

// Status represents the current status of a task
type Status int

const (
	StatusPending Status = iota
	StatusRunning
	StatusCompleted
	StatusFailed
)

// String returns the string representation of the status
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
	default:
		return "unknown"
	}
}

// ParseStatus parses a string into a Status
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
	default:
		return StatusPending
	}
}

// Task represents a task in the scheduler
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
}

// NewTask creates a new task with the given priority and payload
func NewTask(priority Priority, payload json.RawMessage) *Task {
	return &Task{
		ID:        uuid.New().String(),
		Priority:  priority,
		Payload:   payload,
		CreatedAt: time.Now(),
		Status:    StatusPending,
	}
}

// TaskRequest represents a request to create a new task
type TaskRequest struct {
	Priority Priority        `json:"priority" binding:"required"`
	Payload  json.RawMessage `json:"payload" binding:"required"`
}

// TaskResponse represents a response for task operations
type TaskResponse struct {
	Task    *Task  `json:"task,omitempty"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// TasksResponse represents a response containing multiple tasks
type TasksResponse struct {
	Tasks   []*Task `json:"tasks"`
	Success bool    `json:"success"`
	Error   string  `json:"error,omitempty"`
}

// NodeInfo represents information about a node in the cluster
type NodeInfo struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	IsLeader bool      `json:"is_leader"`
	LastSeen time.Time `json:"last_seen"`
	Status   string    `json:"status"`
}

// ClusterInfo represents information about the cluster
type ClusterInfo struct {
	Nodes        []*NodeInfo `json:"nodes"`
	LeaderID     string      `json:"leader_id"`
	TotalTasks   int         `json:"total_tasks"`
	PendingTasks int         `json:"pending_tasks"`
	RunningTasks int         `json:"running_tasks"`
}
