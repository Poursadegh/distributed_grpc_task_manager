package storage

import (
	"context"
	"database/sql"
	"fmt"

	"task-scheduler/internal/types"

	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &PostgresStorage{db: db}, nil
}

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS tasks (
			id VARCHAR(255) PRIMARY KEY,
			priority VARCHAR(50) NOT NULL,
			payload JSONB NOT NULL,
			status VARCHAR(50) NOT NULL,
			created_at TIMESTAMP NOT NULL,
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			worker_id VARCHAR(255),
			error TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS nodes (
			id VARCHAR(255) PRIMARY KEY,
			address VARCHAR(255) NOT NULL,
			last_seen TIMESTAMP NOT NULL,
			is_leader BOOLEAN DEFAULT FALSE
		)`,
		`CREATE TABLE IF NOT EXISTS queue_tasks (
			id VARCHAR(255) PRIMARY KEY,
			task_id VARCHAR(255) NOT NULL,
			priority VARCHAR(50) NOT NULL,
			created_at TIMESTAMP NOT NULL,
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority)`,
		`CREATE INDEX IF NOT EXISTS idx_queue_priority_created ON queue_tasks(priority, created_at)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}

	return nil
}

func (p *PostgresStorage) SaveTask(ctx context.Context, task *types.Task) error {
	query := `
		INSERT INTO tasks (id, priority, payload, status, created_at, started_at, completed_at, worker_id, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			priority = EXCLUDED.priority,
			payload = EXCLUDED.payload,
			status = EXCLUDED.status,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at,
			worker_id = EXCLUDED.worker_id,
			error = EXCLUDED.error
	`

	_, err := p.db.ExecContext(ctx, query,
		task.ID,
		task.Priority.String(),
		task.Payload,
		task.Status.String(),
		task.CreatedAt,
		task.StartedAt,
		task.CompletedAt,
		task.WorkerID,
		task.Error,
	)

	return err
}

func (p *PostgresStorage) GetTask(ctx context.Context, id string) (*types.Task, error) {
	query := `
		SELECT id, priority, payload, status, created_at, started_at, completed_at, worker_id, error
		FROM tasks WHERE id = $1
	`

	var task types.Task
	var priorityStr, statusStr string
	var startedAt, completedAt sql.NullTime

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&priorityStr,
		&task.Payload,
		&statusStr,
		&task.CreatedAt,
		&startedAt,
		&completedAt,
		&task.WorkerID,
		&task.Error,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	task.Priority = types.ParsePriority(priorityStr)
	task.Status = types.ParseStatus(statusStr)

	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}

	return &task, nil
}

func (p *PostgresStorage) UpdateTask(ctx context.Context, task *types.Task) error {
	return p.SaveTask(ctx, task)
}

func (p *PostgresStorage) DeleteTask(ctx context.Context, id string) error {
	query := `DELETE FROM tasks WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgresStorage) GetAllTasks(ctx context.Context) ([]*types.Task, error) {
	query := `
		SELECT id, priority, payload, status, created_at, started_at, completed_at, worker_id, error
		FROM tasks ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*types.Task
	for rows.Next() {
		var task types.Task
		var priorityStr, statusStr string
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(
			&task.ID,
			&priorityStr,
			&task.Payload,
			&statusStr,
			&task.CreatedAt,
			&startedAt,
			&completedAt,
			&task.WorkerID,
			&task.Error,
		)
		if err != nil {
			return nil, err
		}

		task.Priority = types.ParsePriority(priorityStr)
		task.Status = types.ParseStatus(statusStr)

		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

func (p *PostgresStorage) GetTasksByStatus(ctx context.Context, status types.Status) ([]*types.Task, error) {
	query := `
		SELECT id, priority, payload, status, created_at, started_at, completed_at, worker_id, error
		FROM tasks WHERE status = $1 ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, status.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*types.Task
	for rows.Next() {
		var task types.Task
		var priorityStr, statusStr string
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(
			&task.ID,
			&priorityStr,
			&statusStr,
			&task.Payload,
			&task.CreatedAt,
			&startedAt,
			&completedAt,
			&task.WorkerID,
			&task.Error,
		)
		if err != nil {
			return nil, err
		}

		task.Priority = types.ParsePriority(priorityStr)
		task.Status = types.ParseStatus(statusStr)

		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

func (p *PostgresStorage) SaveToQueue(ctx context.Context, task *types.Task) error {
	query := `
		INSERT INTO queue_tasks (id, task_id, priority, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING
	`

	_, err := p.db.ExecContext(ctx, query,
		task.ID,
		task.ID,
		task.Priority.String(),
		task.CreatedAt,
	)

	return err
}

func (p *PostgresStorage) RemoveFromQueue(ctx context.Context, taskID string) error {
	query := `DELETE FROM queue_tasks WHERE task_id = $1`
	_, err := p.db.ExecContext(ctx, query, taskID)
	return err
}

func (p *PostgresStorage) GetQueueTasks(ctx context.Context) ([]*types.Task, error) {
	query := `
		SELECT t.id, t.priority, t.payload, t.status, t.created_at, t.started_at, t.completed_at, t.worker_id, t.error
		FROM queue_tasks q
		JOIN tasks t ON q.task_id = t.id
		ORDER BY q.priority DESC, q.created_at ASC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*types.Task
	for rows.Next() {
		var task types.Task
		var priorityStr, statusStr string
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(
			&task.ID,
			&priorityStr,
			&task.Payload,
			&statusStr,
			&task.CreatedAt,
			&startedAt,
			&completedAt,
			&task.WorkerID,
			&task.Error,
		)
		if err != nil {
			return nil, err
		}

		task.Priority = types.ParsePriority(priorityStr)
		task.Status = types.ParseStatus(statusStr)

		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

func (p *PostgresStorage) SaveNodeInfo(ctx context.Context, node *types.NodeInfo) error {
	query := `
		INSERT INTO nodes (id, address, last_seen, is_leader)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			address = EXCLUDED.address,
			last_seen = EXCLUDED.last_seen,
			is_leader = EXCLUDED.is_leader
	`

	_, err := p.db.ExecContext(ctx, query,
		node.ID,
		node.Address,
		node.LastSeen,
		node.IsLeader,
	)

	return err
}

func (p *PostgresStorage) GetNodeInfo(ctx context.Context, id string) (*types.NodeInfo, error) {
	query := `SELECT id, address, last_seen, is_leader FROM nodes WHERE id = $1`

	var node types.NodeInfo
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&node.ID,
		&node.Address,
		&node.LastSeen,
		&node.IsLeader,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}

	return &node, nil
}

func (p *PostgresStorage) GetAllNodes(ctx context.Context) ([]*types.NodeInfo, error) {
	query := `SELECT id, address, last_seen, is_leader FROM nodes ORDER BY last_seen DESC`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*types.NodeInfo
	for rows.Next() {
		var node types.NodeInfo
		err := rows.Scan(
			&node.ID,
			&node.Address,
			&node.LastSeen,
			&node.IsLeader,
		)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, &node)
	}

	return nodes, nil
}

func (p *PostgresStorage) DeleteNode(ctx context.Context, id string) error {
	query := `DELETE FROM nodes WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgresStorage) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}
