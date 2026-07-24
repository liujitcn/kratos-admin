package codegen

import (
	"context"
	"fmt"
	"sync"
	"time"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/pkg/errorsx"
	transportSSE "github.com/liujitcn/kratos-admin/backend/pkg/sse"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

const (
	// taskRetention 表示已完成代码生成任务的内存保留时间。
	taskRetention = 10 * time.Minute
	// SSEStreamCodeGen 表示管理后台代码生成 SSE 流。
	SSEStreamCodeGen = "system.admin.codegen"
	// SSEEventCodeGenProgress 表示代码生成任务进度事件。
	SSEEventCodeGenProgress = "codegen.progress"
)

// Publisher 将最新任务快照推送给订阅者。
type Publisher func(context.Context, string, *systemadminv1.CodeGenTask)

// Manager 管理不落库的代码生成任务进度。
type Manager struct {
	mu        sync.RWMutex          // 任务和发布器读写锁
	tasks     map[string]*taskEntry // 按任务 ID 保存的内存快照
	publisher Publisher             // SSE 快照发布方法
}

// taskEntry 保存任务所属用户和任务快照。
type taskEntry struct {
	ownerID int64                      // 任务创建用户 ID
	task    *systemadminv1.CodeGenTask // 当前任务快照
}

// NewManager 创建代码生成任务进度管理器。
func NewManager() *Manager {
	return &Manager{tasks: make(map[string]*taskEntry)}
}

// CodeGenSSEStream 解析代码生成任务 SSE 订阅请求。
type CodeGenSSEStream struct {
	manager *Manager
}

var _ transportSSE.Stream = (*CodeGenSSEStream)(nil)

// NewCodeGenSSEStream 创建代码生成任务 SSE 流声明。
func NewCodeGenSSEStream(manager *Manager) *CodeGenSSEStream {
	return &CodeGenSSEStream{manager: manager}
}

// StreamID 返回指定代码生成任务的 SSE 流标识。
func StreamID(taskID string) string {
	return fmt.Sprintf("%s:%s", SSEStreamCodeGen, taskID)
}

// ID 返回代码生成 SSE 流标识。
func (s *CodeGenSSEStream) ID() string {
	return SSEStreamCodeGen
}

// Resolve 校验任务归属后返回隔离的传输流标识。
func (s *CodeGenSSEStream) Resolve(channelID string, userID int64) (string, error) {
	if channelID == "" {
		return "", errorsx.InvalidArgument("代码生成任务ID不能为空")
	}
	if !s.manager.IsOwner(channelID, userID) {
		return "", errorsx.PermissionDenied("无权订阅代码生成任务")
	}
	return StreamID(channelID), nil
}

// SetPublisher 设置任务快照发布方法。
func (m *Manager) SetPublisher(publisher Publisher) {
	m.mu.Lock()
	m.publisher = publisher
	m.mu.Unlock()
}

// Create 创建等待执行的代码生成任务，同一用户同时只允许一个活跃任务。
func (m *Manager) Create(ownerID int64, tables []*systemadminv1.CodeGenTaskTable) (*systemadminv1.CodeGenTask, bool) {
	taskID := uuid.NewString()
	task := &systemadminv1.CodeGenTask{
		TaskId:    taskID,
		Status:    systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_PENDING,
		Message:   "等待执行",
		Tables:    tables,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	recalculateProgress(task)
	m.mu.Lock()
	// 活跃任务按用户互斥，已完成但尚在保留期内的任务不影响再次生成。
	for _, entry := range m.tasks {
		if entry.ownerID == ownerID && !isTerminalTaskStatus(entry.task.Status) {
			m.mu.Unlock()
			return nil, false
		}
	}
	m.tasks[taskID] = &taskEntry{ownerID: ownerID, task: cloneTask(task)}
	m.mu.Unlock()
	return cloneTask(task), true
}

// Snapshot 查询指定用户可访问的任务快照。
func (m *Manager) Snapshot(taskID string, ownerID int64) (*systemadminv1.CodeGenTask, bool) {
	m.mu.RLock()
	entry, ok := m.tasks[taskID]
	if ok && entry.ownerID == ownerID {
		// 持锁期间复制快照，确保调用方看不到并发更新中的中间状态。
		task := cloneTask(entry.task)
		m.mu.RUnlock()
		return task, true
	}
	m.mu.RUnlock()
	return nil, false
}

// IsOwner 判断任务是否属于指定用户。
func (m *Manager) IsOwner(taskID string, ownerID int64) bool {
	m.mu.RLock()
	entry, ok := m.tasks[taskID]
	isOwner := ok && entry.ownerID == ownerID
	m.mu.RUnlock()
	return isOwner
}

// MarkTaskRunning 标记任务开始执行。
func (m *Manager) MarkTaskRunning(ctx context.Context, taskID string) {
	m.update(ctx, taskID, func(task *systemadminv1.CodeGenTask) {
		task.Status = systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_RUNNING
		task.Message = "正在生成代码"
	})
}

// MarkTaskCompleted 标记任务执行结束，并在保留期后释放内存快照。
func (m *Manager) MarkTaskCompleted(ctx context.Context, taskID string, status systemadminv1.CodeGenTaskStatus, message string) {
	m.update(ctx, taskID, func(task *systemadminv1.CodeGenTask) {
		task.Status = status
		task.Message = message
		task.CurrentTableName = ""
		task.FinishedAt = time.Now().Format(time.RFC3339)
	})
	// 终态任务保留一段时间，供刷新页面或重新打开弹窗时恢复最终结果。
	time.AfterFunc(taskRetention, func() {
		m.mu.Lock()
		delete(m.tasks, taskID)
		m.mu.Unlock()
	})
}

// MarkTableRunning 标记单个生成对象开始执行。
func (m *Manager) MarkTableRunning(ctx context.Context, taskID string, tableID int64) {
	m.update(ctx, taskID, func(task *systemadminv1.CodeGenTask) {
		table := findTaskTable(task, tableID)
		if table == nil {
			return
		}
		table.Status = systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_RUNNING
		table.Message = "正在生成"
		task.CurrentTableName = table.TableName
	})
}

// MarkTableCompleted 标记单个生成对象执行结束。
func (m *Manager) MarkTableCompleted(ctx context.Context, taskID string, tableID int64, status systemadminv1.CodeGenTaskStatus, message string) {
	m.update(ctx, taskID, func(task *systemadminv1.CodeGenTask) {
		table := findTaskTable(task, tableID)
		if table == nil {
			return
		}
		table.Status = status
		table.Message = message
		if status == systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED {
			for _, step := range table.Steps {
				if isTerminalStepStatus(step.Status) {
					continue
				}
				step.Status = systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SKIPPED
				step.Message = "生成失败，未继续执行"
			}
		}
	})
}

// RegisterSteps 登记单个生成对象的全部执行步骤。
func (m *Manager) RegisterSteps(ctx context.Context, taskID string, tableID int64, steps []*systemadminv1.CodeGenTaskStep) {
	m.update(ctx, taskID, func(task *systemadminv1.CodeGenTask) {
		table := findTaskTable(task, tableID)
		if table == nil {
			return
		}
		table.Steps = steps
	})
}

// UpdateStep 更新单个生成步骤的状态和结果。
func (m *Manager) UpdateStep(
	ctx context.Context,
	taskID string,
	tableID int64,
	stepID string,
	status systemadminv1.CodeGenTaskStepStatus,
	message string,
	output string,
) {
	m.update(ctx, taskID, func(task *systemadminv1.CodeGenTask) {
		table := findTaskTable(task, tableID)
		if table == nil {
			return
		}
		for _, step := range table.Steps {
			if step.Id != stepID {
				continue
			}
			step.Status = status
			step.Message = message
			step.Output = output
			return
		}
	})
}

// update 修改任务快照，并在解锁后发布不可变副本。
func (m *Manager) update(ctx context.Context, taskID string, update func(*systemadminv1.CodeGenTask)) {
	m.mu.Lock()
	entry := m.tasks[taskID]
	if entry == nil {
		m.mu.Unlock()
		return
	}
	update(entry.task)
	recalculateProgress(entry.task)
	// 发布端只接收深拷贝，不能绕过锁修改管理器内部状态。
	task := cloneTask(entry.task)
	publisher := m.publisher
	m.mu.Unlock()

	// SSE 发布可能阻塞或触发网络 IO，必须在释放任务锁后执行。
	if publisher != nil {
		publisher(ctx, taskID, task)
	}
}

// recalculateProgress 重算单表和整批任务的完成步骤数。
func recalculateProgress(task *systemadminv1.CodeGenTask) {
	task.TotalSteps = 0
	task.CompletedSteps = 0
	for _, table := range task.Tables {
		// 表级进度由终态步骤计算，任务级进度再聚合所有表，避免维护多份计数状态。
		table.TotalSteps = int32(len(table.Steps))
		table.CompletedSteps = 0
		for _, step := range table.Steps {
			if isTerminalStepStatus(step.Status) {
				table.CompletedSteps++
			}
		}
		task.TotalSteps += table.TotalSteps
		task.CompletedSteps += table.CompletedSteps
	}
}

// isTerminalStepStatus 判断步骤是否已经结束。
func isTerminalStepStatus(status systemadminv1.CodeGenTaskStepStatus) bool {
	return status == systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SUCCEEDED ||
		status == systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_FAILED ||
		status == systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SKIPPED
}

// isTerminalTaskStatus 判断任务是否已经结束。
func isTerminalTaskStatus(status systemadminv1.CodeGenTaskStatus) bool {
	return status == systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_SUCCEEDED ||
		status == systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED
}

// findTaskTable 按生成对象 ID 查询任务明细。
func findTaskTable(task *systemadminv1.CodeGenTask, tableID int64) *systemadminv1.CodeGenTaskTable {
	for _, table := range task.Tables {
		if table.TableId == tableID {
			return table
		}
	}
	return nil
}

// cloneTask 复制任务快照，避免调用方与管理器共享可变对象。
func cloneTask(task *systemadminv1.CodeGenTask) *systemadminv1.CodeGenTask {
	return proto.Clone(task).(*systemadminv1.CodeGenTask)
}
