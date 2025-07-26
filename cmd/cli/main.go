package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"task-scheduler/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	var (
		serverAddr = flag.String("server", "localhost:9090", "gRPC server address")
		command    = flag.String("cmd", "stats", "Command to execute (stats, submit, list, health)")
		priority   = flag.String("priority", "medium", "Task priority (high, medium, low)")
		payload    = flag.String("payload", "{}", "Task payload JSON")
		taskID     = flag.String("id", "", "Task ID for get command")
		status     = flag.String("status", "", "Status filter for list command")
	)
	flag.Parse()

	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := proto.NewTaskSchedulerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch *command {
	case "stats":
		showStats(ctx, client)
	case "submit":
		submitTask(ctx, client, *priority, *payload)
	case "list":
		listTasks(ctx, client, *status)
	case "get":
		getTask(ctx, client, *taskID)
	case "health":
		checkHealth(ctx, client)
	case "cluster":
		showClusterInfo(ctx, client)
	default:
		fmt.Printf("Unknown command: %s\n", *command)
		fmt.Println("Available commands: stats, submit, list, get, health, cluster")
	}
}

func showStats(ctx context.Context, client proto.TaskSchedulerClient) {
	resp, err := client.GetStats(ctx, &proto.GetStatsRequest{})
	if err != nil {
		log.Fatalf("Failed to get stats: %v", err)
	}

	if !resp.Success {
		log.Fatalf("Failed to get stats: %s", resp.Error)
	}

	fmt.Println("=== Queue Statistics ===")
	fmt.Printf("Total Tasks: %d\n", resp.QueueStats.TotalTasks)
	fmt.Printf("High Priority: %d\n", resp.QueueStats.HighPriority)
	fmt.Printf("Medium Priority: %d\n", resp.QueueStats.MediumPriority)
	fmt.Printf("Low Priority: %d\n", resp.QueueStats.LowPriority)

	fmt.Println("\n=== Worker Metrics ===")
	fmt.Printf("Tasks Processed: %d\n", resp.WorkerMetrics.TasksProcessed)
	fmt.Printf("Tasks Failed: %d\n", resp.WorkerMetrics.TasksFailed)
	fmt.Printf("Tasks In Progress: %d\n", resp.WorkerMetrics.TasksInProgress)
	fmt.Printf("Worker Count: %d\n", resp.WorkerMetrics.WorkerCount)
	fmt.Printf("Queue Size: %d\n", resp.WorkerMetrics.QueueSize)
}

func submitTask(ctx context.Context, client proto.TaskSchedulerClient, priorityStr, payloadStr string) {
	priority := proto.Priority_PRIORITY_MEDIUM
	switch priorityStr {
	case "high":
		priority = proto.Priority_PRIORITY_HIGH
	case "medium":
		priority = proto.Priority_PRIORITY_MEDIUM
	case "low":
		priority = proto.Priority_PRIORITY_LOW
	}

	resp, err := client.SubmitTask(ctx, &proto.SubmitTaskRequest{
		Priority: priority,
		Payload:  []byte(payloadStr),
	})
	if err != nil {
		log.Fatalf("Failed to submit task: %v", err)
	}

	if !resp.Success {
		log.Fatalf("Failed to submit task: %s", resp.Error)
	}

	fmt.Printf("Task submitted successfully!\n")
	fmt.Printf("Task ID: %s\n", resp.Task.Id)
	fmt.Printf("Priority: %s\n", resp.Task.Priority)
	fmt.Printf("Status: %s\n", resp.Task.Status)
}

func listTasks(ctx context.Context, client proto.TaskSchedulerClient, statusFilter string) {
	var status proto.Status
	if statusFilter != "" {
		switch statusFilter {
		case "pending":
			status = proto.Status_STATUS_PENDING
		case "running":
			status = proto.Status_STATUS_RUNNING
		case "completed":
			status = proto.Status_STATUS_COMPLETED
		case "failed":
			status = proto.Status_STATUS_FAILED
		default:
			log.Fatalf("Invalid status: %s", statusFilter)
		}

		resp, err := client.GetTasksByStatus(ctx, &proto.GetTasksByStatusRequest{
			Status: status,
		})
		if err != nil {
			log.Fatalf("Failed to get tasks by status: %v", err)
		}

		if !resp.Success {
			log.Fatalf("Failed to get tasks: %s", resp.Error)
		}

		fmt.Printf("Tasks with status '%s':\n", statusFilter)
		for _, task := range resp.Tasks {
			printTask(task)
		}
	} else {
		resp, err := client.ListTasks(ctx, &proto.ListTasksRequest{})
		if err != nil {
			log.Fatalf("Failed to list tasks: %v", err)
		}

		if !resp.Success {
			log.Fatalf("Failed to list tasks: %s", resp.Error)
		}

		fmt.Println("All tasks:")
		for _, task := range resp.Tasks {
			printTask(task)
		}
	}
}

func getTask(ctx context.Context, client proto.TaskSchedulerClient, taskID string) {
	if taskID == "" {
		log.Fatal("Task ID is required")
	}

	resp, err := client.GetTask(ctx, &proto.GetTaskRequest{Id: taskID})
	if err != nil {
		log.Fatalf("Failed to get task: %v", err)
	}

	if !resp.Success {
		log.Fatalf("Failed to get task: %s", resp.Error)
	}

	printTask(resp.Task)
}

func checkHealth(ctx context.Context, client proto.TaskSchedulerClient) {
	resp, err := client.HealthCheck(ctx, &proto.HealthCheckRequest{})
	if err != nil {
		log.Fatalf("Failed to check health: %v", err)
	}

	fmt.Printf("Health Check:\n")
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Node ID: %s\n", resp.NodeId)
	fmt.Printf("Is Leader: %t\n", resp.IsLeader)
	fmt.Printf("Timestamp: %d\n", resp.Timestamp)
}

func showClusterInfo(ctx context.Context, client proto.TaskSchedulerClient) {
	resp, err := client.GetClusterInfo(ctx, &proto.GetClusterInfoRequest{})
	if err != nil {
		log.Fatalf("Failed to get cluster info: %v", err)
	}

	if !resp.Success {
		log.Fatalf("Failed to get cluster info: %s", resp.Error)
	}

	fmt.Println("=== Cluster Information ===")
	fmt.Printf("Leader ID: %s\n", resp.Info.LeaderId)
	fmt.Printf("Total Tasks: %d\n", resp.Info.TotalTasks)
	fmt.Printf("Pending Tasks: %d\n", resp.Info.PendingTasks)
	fmt.Printf("Running Tasks: %d\n", resp.Info.RunningTasks)

	fmt.Println("\n=== Nodes ===")
	for _, node := range resp.Info.Nodes {
		fmt.Printf("Node ID: %s\n", node.Id)
		fmt.Printf("  Address: %s\n", node.Address)
		fmt.Printf("  Is Leader: %t\n", node.IsLeader)
		fmt.Printf("  Status: %s\n", node.Status)
		fmt.Printf("  Last Seen: %d\n", node.LastSeen)
		fmt.Println()
	}
}

func printTask(task *proto.Task) {
	fmt.Printf("Task ID: %s\n", task.Id)
	fmt.Printf("  Priority: %s\n", task.Priority)
	fmt.Printf("  Status: %s\n", task.Status)
	fmt.Printf("  Created At: %d\n", task.CreatedAt)
	if task.StartedAt > 0 {
		fmt.Printf("  Started At: %d\n", task.StartedAt)
	}
	if task.CompletedAt > 0 {
		fmt.Printf("  Completed At: %d\n", task.CompletedAt)
	}
	if task.WorkerId != "" {
		fmt.Printf("  Worker ID: %s\n", task.WorkerId)
	}
	if task.Error != "" {
		fmt.Printf("  Error: %s\n", task.Error)
	}
	fmt.Printf("  Payload: %s\n", string(task.Payload))
	fmt.Println()
}
