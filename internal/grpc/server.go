package grpc

import (
	"context"
	"log"
	"net"
	"time"

	"task-scheduler/internal/scheduler"
	"task-scheduler/internal/types"
	"task-scheduler/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	proto.UnimplementedTaskSchedulerServer
	scheduler *scheduler.Scheduler
	server    *grpc.Server
}

func NewServer(scheduler *scheduler.Scheduler) *Server {
	return &Server{
		scheduler: scheduler,
		server:    grpc.NewServer(),
	}
}

func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	proto.RegisterTaskSchedulerServer(s.server, s)
	reflection.Register(s.server)

	log.Printf("gRPC server starting on %s", addr)
	return s.server.Serve(lis)
}

func (s *Server) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}

func (s *Server) SubmitTask(ctx context.Context, req *proto.SubmitTaskRequest) (*proto.SubmitTaskResponse, error) {
	priority := convertPriority(req.Priority)
	task, err := s.scheduler.SubmitTask(priority, req.Payload)
	if err != nil {
		return &proto.SubmitTaskResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &proto.SubmitTaskResponse{
		Task:    convertTask(task),
		Success: true,
	}, nil
}

func (s *Server) GetTask(ctx context.Context, req *proto.GetTaskRequest) (*proto.GetTaskResponse, error) {
	task, err := s.scheduler.GetTask(req.Id)
	if err != nil {
		return &proto.GetTaskResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	if task == nil {
		return &proto.GetTaskResponse{
			Success: false,
			Error:   "task not found",
		}, nil
	}

	return &proto.GetTaskResponse{
		Task:    convertTask(task),
		Success: true,
	}, nil
}

func (s *Server) ListTasks(ctx context.Context, req *proto.ListTasksRequest) (*proto.ListTasksResponse, error) {
	tasks, err := s.scheduler.GetAllTasks()
	if err != nil {
		return &proto.ListTasksResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	protoTasks := make([]*proto.Task, len(tasks))
	for i, task := range tasks {
		protoTasks[i] = convertTask(task)
	}

	return &proto.ListTasksResponse{
		Tasks:   protoTasks,
		Success: true,
	}, nil
}

func (s *Server) GetTasksByStatus(ctx context.Context, req *proto.GetTasksByStatusRequest) (*proto.ListTasksResponse, error) {
	status := convertStatus(req.Status)
	tasks, err := s.scheduler.GetTasksByStatus(status)
	if err != nil {
		return &proto.ListTasksResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	protoTasks := make([]*proto.Task, len(tasks))
	for i, task := range tasks {
		protoTasks[i] = convertTask(task)
	}

	return &proto.ListTasksResponse{
		Tasks:   protoTasks,
		Success: true,
	}, nil
}

func (s *Server) GetClusterInfo(ctx context.Context, req *proto.GetClusterInfoRequest) (*proto.GetClusterInfoResponse, error) {
	info := s.scheduler.GetClusterInfo()
	return &proto.GetClusterInfoResponse{
		Info:    convertClusterInfo(info),
		Success: true,
	}, nil
}

func (s *Server) GetStats(ctx context.Context, req *proto.GetStatsRequest) (*proto.GetStatsResponse, error) {
	queueStats := s.scheduler.GetQueueStats()
	workerMetrics := s.scheduler.GetWorkerMetrics()

	return &proto.GetStatsResponse{
		QueueStats: &proto.QueueStats{
			TotalTasks:     int32(queueStats["total_tasks"].(int)),
			HighPriority:   int32(queueStats["high_priority"].(int)),
			MediumPriority: int32(queueStats["medium_priority"].(int)),
			LowPriority:    int32(queueStats["low_priority"].(int)),
		},
		WorkerMetrics: &proto.WorkerMetrics{
			TasksProcessed:  workerMetrics.TasksProcessed,
			TasksFailed:     workerMetrics.TasksFailed,
			TasksInProgress: workerMetrics.TasksInProgress,
			WorkerCount:     int32(workerMetrics.WorkerCount),
			QueueSize:       int32(workerMetrics.QueueSize),
		},
		Success: true,
	}, nil
}

func (s *Server) HealthCheck(ctx context.Context, req *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	return &proto.HealthCheckResponse{
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
		NodeId:    s.scheduler.GetNodeID(),
		IsLeader:  s.scheduler.IsLeader(),
	}, nil
}

func convertPriority(p proto.Priority) types.Priority {
	switch p {
	case proto.Priority_PRIORITY_HIGH:
		return types.PriorityHigh
	case proto.Priority_PRIORITY_MEDIUM:
		return types.PriorityMedium
	case proto.Priority_PRIORITY_LOW:
		return types.PriorityLow
	default:
		return types.PriorityLow
	}
}

func convertStatus(s proto.Status) types.Status {
	switch s {
	case proto.Status_STATUS_PENDING:
		return types.StatusPending
	case proto.Status_STATUS_RUNNING:
		return types.StatusRunning
	case proto.Status_STATUS_COMPLETED:
		return types.StatusCompleted
	case proto.Status_STATUS_FAILED:
		return types.StatusFailed
	default:
		return types.StatusPending
	}
}

func convertTask(task *types.Task) *proto.Task {
	if task == nil {
		return nil
	}

	protoTask := &proto.Task{
		Id:        task.ID,
		Priority:  convertProtoPriority(task.Priority),
		Payload:   task.Payload,
		CreatedAt: task.CreatedAt.Unix(),
		Status:    convertProtoStatus(task.Status),
		WorkerId:  task.WorkerID,
		Error:     task.Error,
	}

	if task.StartedAt != nil {
		protoTask.StartedAt = task.StartedAt.Unix()
	}
	if task.CompletedAt != nil {
		protoTask.CompletedAt = task.CompletedAt.Unix()
	}

	return protoTask
}

func convertProtoPriority(p types.Priority) proto.Priority {
	switch p {
	case types.PriorityHigh:
		return proto.Priority_PRIORITY_HIGH
	case types.PriorityMedium:
		return proto.Priority_PRIORITY_MEDIUM
	case types.PriorityLow:
		return proto.Priority_PRIORITY_LOW
	default:
		return proto.Priority_PRIORITY_LOW
	}
}

func convertProtoStatus(s types.Status) proto.Status {
	switch s {
	case types.StatusPending:
		return proto.Status_STATUS_PENDING
	case types.StatusRunning:
		return proto.Status_STATUS_RUNNING
	case types.StatusCompleted:
		return proto.Status_STATUS_COMPLETED
	case types.StatusFailed:
		return proto.Status_STATUS_FAILED
	default:
		return proto.Status_STATUS_PENDING
	}
}

func convertClusterInfo(info *types.ClusterInfo) *proto.ClusterInfo {
	if info == nil {
		return nil
	}

	nodes := make([]*proto.NodeInfo, len(info.Nodes))
	for i, node := range info.Nodes {
		nodes[i] = &proto.NodeInfo{
			Id:       node.ID,
			Address:  node.Address,
			IsLeader: node.IsLeader,
			LastSeen: node.LastSeen.Unix(),
			Status:   node.Status,
		}
	}

	return &proto.ClusterInfo{
		Nodes:        nodes,
		LeaderId:     info.LeaderID,
		TotalTasks:   int32(info.TotalTasks),
		PendingTasks: int32(info.PendingTasks),
		RunningTasks: int32(info.RunningTasks),
	}
}
