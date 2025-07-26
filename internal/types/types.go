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
