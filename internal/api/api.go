package api

import (
	"net/http"
	"time"

	"task-scheduler/internal/scheduler"
	"task-scheduler/internal/types"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type API struct {
	scheduler *scheduler.Scheduler
	router    *gin.Engine

	tasksSubmitted prometheus.Counter
	tasksCompleted prometheus.Counter
	tasksFailed    prometheus.Counter
	taskDuration   prometheus.Histogram
	queueSize      prometheus.Gauge
	workerCount    prometheus.Gauge
}

func NewAPI(scheduler *scheduler.Scheduler) *API {
	api := &API{
		scheduler: scheduler,
		router:    gin.Default(),
	}

	api.initMetrics()

	api.setupRoutes()

	return api
}

func (api *API) initMetrics() {
	api.tasksSubmitted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tasks_submitted_total",
		Help: "Total number of tasks submitted",
	})

	api.tasksCompleted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tasks_completed_total",
		Help: "Total number of tasks completed",
	})

	api.tasksFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tasks_failed_total",
		Help: "Total number of tasks failed",
	})

	api.taskDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "task_duration_seconds",
		Help:    "Task duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	api.queueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "queue_size",
		Help: "Current number of tasks in queue",
	})

	api.workerCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "worker_count",
		Help: "Current number of active workers",
	})

	prometheus.MustRegister(api.tasksSubmitted)
	prometheus.MustRegister(api.tasksCompleted)
	prometheus.MustRegister(api.tasksFailed)
	prometheus.MustRegister(api.taskDuration)
	prometheus.MustRegister(api.queueSize)
	prometheus.MustRegister(api.workerCount)
}

func (api *API) setupRoutes() {
	api.router.GET("/health", api.healthHandler)

	tasks := api.router.Group("/api/v1/tasks")
	{
		tasks.POST("/", api.submitTask)
		tasks.GET("/", api.listTasks)
		tasks.GET("/:id", api.getTask)
		tasks.GET("/status/:status", api.getTasksByStatus)
	}

	cluster := api.router.Group("/api/v1/cluster")
	{
		cluster.GET("/info", api.getClusterInfo)
		cluster.GET("/stats", api.getStats)
	}

	api.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

func (api *API) Run(addr string) error {
	return api.router.Run(addr)
}

func (api *API) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"node_id":   api.scheduler.GetNodeID(),
		"is_leader": api.scheduler.IsLeader(),
	})
}

func (api *API) submitTask(c *gin.Context) {
	var req types.TaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body: " + err.Error(),
		})
		return
	}

	task, err := api.scheduler.SubmitTask(req.Priority, req.Payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	api.tasksSubmitted.Inc()

	c.JSON(http.StatusCreated, types.TaskResponse{
		Task:    task,
		Success: true,
	})
}

func (api *API) listTasks(c *gin.Context) {
	tasks, err := api.scheduler.GetAllTasks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.TasksResponse{
		Tasks:   tasks,
		Success: true,
	})
}

func (api *API) getTask(c *gin.Context) {
	taskID := c.Param("id")

	task, err := api.scheduler.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Task not found",
		})
		return
	}

	c.JSON(http.StatusOK, types.TaskResponse{
		Task:    task,
		Success: true,
	})
}

func (api *API) getTasksByStatus(c *gin.Context) {
	statusStr := c.Param("status")
	status := types.ParseStatus(statusStr)

	tasks, err := api.scheduler.GetTasksByStatus(status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.TasksResponse{
		Tasks:   tasks,
		Success: true,
	})
}

func (api *API) getClusterInfo(c *gin.Context) {
	info := api.scheduler.GetClusterInfo()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    info,
	})
}

func (api *API) getStats(c *gin.Context) {
	queueStats := api.scheduler.GetQueueStats()
	workerMetrics := api.scheduler.GetWorkerMetrics()

	api.queueSize.Set(float64(queueStats["total_tasks"].(int)))
	api.workerCount.Set(float64(workerMetrics.WorkerCount))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"queue":  queueStats,
			"worker": workerMetrics,
		},
	})
}
