package codegen

import (
	"context"
	"fmt"
	"sync"
	"time"
	"uuid"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
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

// Publisher 发布 SSE 结构化事件。
type Publisher func(ctx context.Context, streamID, eventID string, payload any)

// Manager 管理不落库的代码生成任务进度。
type Manager struct {
	mu        sync.RWMutex          // 任务读写锁
	tasks     map[string]*taskEntry // 按任务 ID 保存的内存快照
	publisher Publisher             // SSE 事件发布能力
}

// taskEntry 保存任务所属用户和任务快照。
type taskEntry struct {
	ownerID     int64                // 任务创建用户 ID
	localeState LocaleState          // 创建任务时的语言状态
	task        *adminv1.CodeGenTask // 当前任务快照
}

// NewManager 创建代码生成任务进度管理器。
func NewManager() *Manager {
	return &Manager{
		tasks: make(map[string]*taskEntry),
	}
}

// SetPublisher 设置代码生成任务的 SSE 发布能力。
func (m *Manager) SetPublisher(publisher Publisher) {
	m.mu.Lock()
	m.publisher = publisher
	m.mu.Unlock()
}

// StreamID 返回指定代码生成任务的 SSE 流标识。
func StreamID(taskID string) string {
	return fmt.Sprintf("%s:%s", SSEStreamCodeGen, taskID)
}

// Create 创建等待执行的代码生成任务，同一用户同时只允许一个活跃任务。
func (m *Manager) Create(ownerID int64, tables []*adminv1.CodeGenTaskTable, localeState LocaleState) (*adminv1.CodeGenTask, bool) {
	taskID := uuid.New().String()
	task := &adminv1.CodeGenTask{
		TaskId:    taskID,
		Status:    adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_PENDING,
		Message:   Message(localeState, "progress.pending_execute", nil),
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
	m.tasks[taskID] = &taskEntry{ownerID: ownerID, localeState: localeState, task: cloneTask(task)}
	m.mu.Unlock()
	return cloneTask(task), true
}

// Snapshot 查询指定用户可访问的任务快照。
func (m *Manager) Snapshot(taskID string, ownerID int64) (*adminv1.CodeGenTask, bool) {
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

// Message 返回任务创建语言对应的代码生成文案。
func (m *Manager) Message(taskID string, key string, values map[string]string) string {
	m.mu.RLock()
	entry := m.tasks[taskID]
	var localeState LocaleState
	if entry != nil {
		localeState = entry.localeState
	}
	m.mu.RUnlock()
	return Message(localeState, key, values)
}

// MarkTaskRunning 标记任务开始执行。
func (m *Manager) MarkTaskRunning(ctx context.Context, taskID string) {
	m.update(ctx, taskID, func(task *adminv1.CodeGenTask, localeState LocaleState) {
		task.Status = adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_RUNNING
		task.Message = Message(localeState, "progress.running_code_generation", nil)
	})
}

// MarkTaskCompleted 标记任务执行结束，并在保留期后释放内存快照。
func (m *Manager) MarkTaskCompleted(ctx context.Context, taskID string, status adminv1.CodeGenTaskStatus, message string) {
	m.update(ctx, taskID, func(task *adminv1.CodeGenTask, _ LocaleState) {
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
	m.update(ctx, taskID, func(task *adminv1.CodeGenTask, localeState LocaleState) {
		table := findTaskTable(task, tableID)
		if table == nil {
			return
		}
		table.Status = adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_RUNNING
		table.Message = Message(localeState, "progress.running_table", nil)
		task.CurrentTableName = table.TableName
	})
}

// MarkTableCompleted 标记单个生成对象执行结束。
func (m *Manager) MarkTableCompleted(ctx context.Context, taskID string, tableID int64, status adminv1.CodeGenTaskStatus, message string) {
	m.update(ctx, taskID, func(task *adminv1.CodeGenTask, localeState LocaleState) {
		table := findTaskTable(task, tableID)
		if table == nil {
			return
		}
		table.Status = status
		table.Message = message
		if status == adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED {
			for _, step := range table.Steps {
				if isTerminalStepStatus(step.Status) {
					continue
				}
				step.Status = adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SKIPPED
				step.Message = Message(localeState, "progress.generation_failed_skipped", nil)
			}
		}
	})
}

// RegisterSteps 登记单个生成对象的全部执行步骤。
func (m *Manager) RegisterSteps(ctx context.Context, taskID string, tableID int64, steps []*adminv1.CodeGenTaskStep) {
	m.update(ctx, taskID, func(task *adminv1.CodeGenTask, _ LocaleState) {
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
	status adminv1.CodeGenTaskStepStatus,
	message string,
	output string,
) {
	m.update(ctx, taskID, func(task *adminv1.CodeGenTask, _ LocaleState) {
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
func (m *Manager) update(ctx context.Context, taskID string, update func(*adminv1.CodeGenTask, LocaleState)) {
	m.mu.Lock()
	entry := m.tasks[taskID]
	if entry == nil {
		m.mu.Unlock()
		return
	}
	update(entry.task, entry.localeState)
	recalculateProgress(entry.task)
	// 发布端只接收深拷贝，不能绕过锁修改管理器内部状态。
	task := cloneTask(entry.task)
	publisher := m.publisher
	m.mu.Unlock()

	// SSE 发布可能阻塞或触发网络 IO，必须在释放任务锁后执行。
	if publisher != nil {
		publisher(ctx, StreamID(taskID), SSEEventCodeGenProgress, task)
	}
}

// recalculateProgress 重算单表和整批任务的完成步骤数。
func recalculateProgress(task *adminv1.CodeGenTask) {
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
func isTerminalStepStatus(status adminv1.CodeGenTaskStepStatus) bool {
	return status == adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SUCCEEDED ||
		status == adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_FAILED ||
		status == adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SKIPPED
}

// isTerminalTaskStatus 判断任务是否已经结束。
func isTerminalTaskStatus(status adminv1.CodeGenTaskStatus) bool {
	return status == adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_SUCCEEDED ||
		status == adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED
}

// findTaskTable 按生成对象 ID 查询任务明细。
func findTaskTable(task *adminv1.CodeGenTask, tableID int64) *adminv1.CodeGenTaskTable {
	for _, table := range task.Tables {
		if table.TableId == tableID {
			return table
		}
	}
	return nil
}

// cloneTask 复制任务快照，避免调用方与管理器共享可变对象。
func cloneTask(task *adminv1.CodeGenTask) *adminv1.CodeGenTask {
	return proto.Clone(task).(*adminv1.CodeGenTask)
}
