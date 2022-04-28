package task

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/filewriter"
	"github.com/aws/eks-anywhere/pkg/logger"
	"github.com/aws/eks-anywhere/pkg/providers"
	"github.com/aws/eks-anywhere/pkg/types"
	"github.com/aws/eks-anywhere/pkg/workflows/interfaces"
)

const checkpointFileName = "checkpoint.yaml"

// Task is a logical unit of work - meant to be implemented by each Task
type Task interface {
	Run(ctx context.Context, commandContext *CommandContext) Task
	Name() string
	Checkpoint() TaskCheckpoint
	// Restores the command context from a saved checkpoint
	// if this task was already completed and returns the next task
	Restore(ctx context.Context, unmarshalCheckpoint UnmarshallTaskCheckpoint, commandContext *CommandContext) (Task, error)
}

type BasicTask struct{}

func (BasicTask) Checkpoint() TaskCheckpoint {
	return nil
}

func (BasicTask) Restore(ctx context.Context, unmarshalCheckpoint UnmarshallTaskCheckpoint, commandContext *CommandContext) (Task, error) {
	return nil, nil
}

// Command context maintains the mutable and shared entities
type CommandContext struct {
	Bootstrapper       interfaces.Bootstrapper
	Provider           providers.Provider
	ClusterManager     interfaces.ClusterManager
	AddonManager       interfaces.AddonManager
	Validations        interfaces.Validator
	Writer             filewriter.FileWriter
	EksdInstaller      interfaces.EksdInstaller
	EksdUpgrader       interfaces.EksdUpgrader
	CAPIManager        interfaces.CAPIManager
	ClusterSpec        *cluster.Spec
	CurrentClusterSpec *cluster.Spec
	UpgradeChangeDiff  *types.ChangeDiff
	BootstrapCluster   *types.Cluster
	ManagementCluster  *types.Cluster
	WorkloadCluster    *types.Cluster
	Profiler           *Profiler
	OriginalError      error
}

func (c *CommandContext) SetError(err error) {
	if c.OriginalError == nil {
		c.OriginalError = err
	}
}

type Profiler struct {
	metrics map[string]map[string]time.Duration
	starts  map[string]map[string]time.Time
}

// profiler for a Task
func (pp *Profiler) SetStartTask(taskName string) {
	pp.SetStart(taskName, taskName)
}

// this can be used to profile sub tasks
func (pp *Profiler) SetStart(taskName string, msg string) {
	if _, ok := pp.starts[taskName]; !ok {
		pp.starts[taskName] = map[string]time.Time{}
	}
	pp.starts[taskName][msg] = time.Now()
}

// needs to be called after setStart
func (pp *Profiler) MarkDoneTask(taskName string) {
	pp.MarkDone(taskName, taskName)
}

// this can be used to profile sub tasks
func (pp *Profiler) MarkDone(taskName string, msg string) {
	if _, ok := pp.metrics[taskName]; !ok {
		pp.metrics[taskName] = map[string]time.Duration{}
	}
	if start, ok := pp.starts[taskName][msg]; ok {
		pp.metrics[taskName][msg] = time.Since(start)
	}
}

// get Metrics
func (pp *Profiler) Metrics() map[string]map[string]time.Duration {
	return pp.metrics
}

// debug logs for task metric
func (pp *Profiler) logProfileSummary(taskName string) {
	if durationMap, ok := pp.metrics[taskName]; ok {
		for k, v := range durationMap {
			if k != taskName {
				logger.V(4).Info("Subtask finished", "task_name", taskName, "subtask_name", k, "duration", v)
			}
		}
		if totalTaskDuration, ok := durationMap[taskName]; ok {
			logger.V(4).Info("Task finished", "task_name", taskName, "duration", totalTaskDuration)
			logger.V(4).Info("----------------------------------")
		}
	}
}

// Manages Task execution
type taskRunner struct {
	writer           filewriter.FileWriter
	firstTask        Task
	singleTaskRunner singleTaskRunner
}

type singleTaskRunner func(ctx context.Context, commandContext *CommandContext, task Task) Task

// executes Task
func (pr *taskRunner) RunTask(ctx context.Context, commandContext *CommandContext) error {
	commandContext.Profiler = &Profiler{
		metrics: make(map[string]map[string]time.Duration),
		starts:  make(map[string]map[string]time.Time),
	}
	task := pr.firstTask
	start := time.Now()
	defer taskRunnerFinalBlock(start)
	checkpointInfo := newCheckpointInfo()
	for task != nil {
		logger.V(4).Info("Task start", "task_name", task.Name())
		commandContext.Profiler.SetStartTask(task.Name())
		nextTask := pr.singleTaskRunner(ctx, commandContext, task)
		commandContext.Profiler.MarkDoneTask(task.Name())
		commandContext.Profiler.logProfileSummary(task.Name())
		if commandContext.OriginalError == nil {
			checkpointInfo.taskCompleted(task.Name(), task.Checkpoint())
		}
		task = nextTask
	}

	if commandContext.OriginalError != nil {
		pr.saveCheckpoint(checkpointInfo)
	}

	return commandContext.OriginalError
}

func (pr *taskRunner) saveCheckpoint(checkpointInfo checkpointInfo) {
	log.Printf("Saving checkpoint:\n%v\n", checkpointInfo)
	content, err := yaml.Marshal(checkpointInfo)
	if err != nil {
		log.Printf("failed saving task runner checkpoint: %v\n", err)
	}

	if _, err = pr.writer.Write(checkpointFileName, content); err != nil {
		log.Printf("failed saving task runner checkpoint: %v\n", err)
	}
}

func runSimpleTask(ctx context.Context, commandContext *CommandContext, task Task) Task {
	return task.Run(ctx, commandContext)
}

func taskRunnerFinalBlock(startTime time.Time) {
	logger.V(4).Info("Tasks completed", "duration", time.Since(startTime))
}

type TaskRunnerOpt func(*taskRunner)

func WithCheckpointFile(fileDir string) TaskRunnerOpt {
	file := filepath.Join(fileDir, checkpointFileName)
	return func(t *taskRunner) {
		logger.Info("Reading checkpoint", "file", file)
		content, err := os.ReadFile(file)
		if err != nil {
			log.Printf("failed reading checkpoint file: %v\n", err)
		}
		checkpointInfo := &checkpointInfo{}
		err = yaml.Unmarshal(content, checkpointInfo)
		if err != nil {
			log.Printf("failed unmarshalling checkpoint: %v\n", err)
		}

		t.singleTaskRunner = newCheckpointTaskRunner(*checkpointInfo).run
	}
}

func NewTaskRunner(task Task, writer filewriter.FileWriter, opts ...TaskRunnerOpt) *taskRunner {
	t := &taskRunner{
		firstTask:        task,
		singleTaskRunner: runSimpleTask,
		writer:           writer,
	}

	for _, o := range opts {
		o(t)
	}

	return t
}

type TaskCheckpoint interface{}

type UnmarshallTaskCheckpoint func(config interface{}) error

type checkpointInfo struct {
	CompletedTasks map[string]TaskCheckpoint `json:"completedTasks"`
}

func newCheckpointInfo() checkpointInfo {
	return checkpointInfo{
		CompletedTasks: map[string]TaskCheckpoint{},
	}
}

func (c checkpointInfo) taskCompleted(name string, checkpoint TaskCheckpoint) {
	c.CompletedTasks[name] = checkpoint
}

func newCheckpointTaskRunner(checkpoint checkpointInfo) checkpointTaskRunner {
	return checkpointTaskRunner{
		checkpoint: checkpoint,
	}
}

type checkpointTaskRunner struct {
	checkpoint checkpointInfo
}

func (c checkpointTaskRunner) run(ctx context.Context, commandContext *CommandContext, task Task) Task {
	taskCheckpoint, ok := c.checkpoint.CompletedTasks[task.Name()]
	if !ok {
		return task.Run(ctx, commandContext)
	}

	nextTask, err := task.Restore(ctx, newUnmarshallTaskCheckpoint(taskCheckpoint), commandContext)
	if err != nil {
		commandContext.SetError(err)
		return nil
	}
	logger.V(4).Info("Task restored from checkpoint", "task_name", task.Name())

	return nextTask
}

func newUnmarshallTaskCheckpoint(taskCheckpoint TaskCheckpoint) UnmarshallTaskCheckpoint {
	return func(config interface{}) error {
		// TODO: inefficient
		checkpointYaml, err := yaml.Marshal(taskCheckpoint)
		if err != nil {
			return nil
		}

		return yaml.Unmarshal(checkpointYaml, config)
	}
}
