package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/codegen"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/migration"
	"github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	corei18n "github.com/liujitcn/kratos-core/resource/i18n"

	"github.com/liujitcn/go-utils/stringcase"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gen/field"
)

var codeGenGenerationProcessLock sync.Mutex

// codeGenProgressReporter 将单个生成对象的执行步骤写入内存任务。
type codeGenProgressReporter struct {
	manager *codegen.Manager // 内存任务管理器
	taskID  string           // 批量生成任务ID
	tableID int64            // 当前生成对象ID
}

// codeGenCommandTarget 描述批量生成命令关联的业务表和进度上报器。
type codeGenCommandTarget struct {
	tableID   int64
	tableName string
	progress  *codeGenProgressReporter
}

// codeGenCommandResult 保存单个业务表在共享命令链中的最终结果。
type codeGenCommandResult struct {
	message string
	err     error
}

// codeGenBatchContext 保存批次预检后的生成计划和字段快照。
type codeGenBatchContext struct {
	plan           *codegen.BatchGeneration
	columnsByTable map[int64][]*codegen.CodeGenColumn
	localeState    codegen.LocaleState
}

// codeGenFileSnapshot 保存生成事务开始前的单个文件状态。
type codeGenFileSnapshot struct {
	fullPath string
	exists   bool
	content  []byte
	mode     os.FileMode
}

// codeGenFileTransaction 保存本批写入文件的快照，用于数据库或文件失败时恢复工作区。
type codeGenFileTransaction struct {
	snapshots    map[string]codeGenFileSnapshot
	writtenPaths []string
	committed    bool
}

const codeGenRestoreRoot = "backend/data/codegen/restore"

// codeGenRestoreManifest 保存一次生成前的文件和菜单快照。
type codeGenRestoreManifest struct {
	Version          int                  `json:"version"`
	TaskID           string               `json:"task_id"`
	TableID          int64                `json:"table_id"`
	BatchTableIDs    []int64              `json:"batch_table_ids"`
	Files            []codeGenRestoreFile `json:"files"`
	Menus            []*models.BaseMenu   `json:"menus,omitempty"`
	GeneratedMenuIDs []int64              `json:"generated_menu_ids,omitempty"`
}

// codeGenRestoreFile 保存单个文件生成前后的内容。
type codeGenRestoreFile struct {
	Path             string  `json:"path"`
	OriginalExists   bool    `json:"original_exists"`
	OriginalContent  string  `json:"original_content"`
	OriginalMode     uint32  `json:"original_mode"`
	GeneratedExists  bool    `json:"generated_exists"`
	GeneratedContent string  `json:"generated_content"`
	OwnerTableIDs    []int64 `json:"owner_table_ids"`
}

// codeGenRestoreWorkspaceFile 表示工作区文件的当前快照。
type codeGenRestoreWorkspaceFile struct {
	Path    string
	Exists  bool
	Content []byte
	Mode    os.FileMode
}

// codeGenRestoreTransaction 管理还原快照文件的失败回滚。
type codeGenRestoreTransaction struct {
	snapshots    map[string]codeGenFileSnapshot
	writtenPaths []string
	committed    bool
}

// CodeGenCase 管理代码预览、批量生成与任务进度。
type CodeGenCase struct {
	*biz.BaseCase
	tx                data.Transaction
	baseAPICase       *BaseAPICase
	codeGenTableCase  *CodeGenTableCase
	codeGenColumnCase *CodeGenColumnCase
	codeGenProtoCase  *CodeGenProtoCase
	baseMenuCase      *BaseMenuCase
	baseMigrationCase *BaseMigrationCase
	baseLanguageCase  *BaseLanguageCase
	progressManager   *codegen.Manager
}

// NewCodeGenCase 创建代码生成执行业务实例。
func NewCodeGenCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseAPICase *BaseAPICase,
	codeGenTableCase *CodeGenTableCase,
	codeGenColumnCase *CodeGenColumnCase,
	codeGenProtoCase *CodeGenProtoCase,
	baseMenuCase *BaseMenuCase,
	baseMigrationCase *BaseMigrationCase,
	baseLanguageCase *BaseLanguageCase,
	catalog *corei18n.I18n,
	progressManager *codegen.Manager,
) *CodeGenCase {
	codegen.SetCatalog(catalog)
	return &CodeGenCase{
		BaseCase:          baseCase,
		tx:                tx,
		baseAPICase:       baseAPICase,
		codeGenTableCase:  codeGenTableCase,
		codeGenColumnCase: codeGenColumnCase,
		codeGenProtoCase:  codeGenProtoCase,
		baseMenuCase:      baseMenuCase,
		baseMigrationCase: baseMigrationCase,
		baseLanguageCase:  baseLanguageCase,
		progressManager:   progressManager,
	}
}

// GetCodeGenTask 查询当前用户可访问的生成任务快照。
func (c *CodeGenCase) GetCodeGenTask(ctx context.Context, taskID string) (*adminv1.CodeGenTask, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	task, ok := c.progressManager.Snapshot(taskID, authInfo.UserId)
	if !ok {
		return &adminv1.CodeGenTask{}, nil
	}
	return task, nil
}

// PreviewCodeGen 根据现有表、字段和Proto配置预览生成结果。
func (c *CodeGenCase) PreviewCodeGen(ctx context.Context, tableID int64, requestedPaths *adminv1.CodeGenOutputPaths) (*adminv1.PreviewCodeGenResponse, error) {
	table, columns, protos, err := c.loadCodeGenContext(ctx, tableID)
	if err != nil {
		return nil, err
	}
	localeState, err := c.baseLanguageCase.LocaleState(ctx)
	if err != nil {
		return nil, err
	}
	var migrationVersion string
	migrationVersion, err = c.latestMigrationVersion(ctx)
	if err != nil {
		return nil, err
	}
	var generation *codegen.Generation
	generation, err = codegen.PrepareGenerationWithMigrationVersion(
		table,
		columns,
		protos,
		requestedPaths,
		table.TableComment,
		migrationVersion,
		codegen.LocaleState{Current: localeState.Current, Enabled: localeState.Enabled, Primary: localeState.Primary},
	)
	if err != nil {
		return nil, err
	}
	if err = c.validateGeneratedBaseAPIs(ctx, generation); err != nil {
		return nil, err
	}
	if err = c.validateGeneratedOptionMethods(ctx, generation.Table, columns, generation.GeneratedMethods); err != nil {
		return nil, err
	}
	if err = c.validateCodeGenParentMenu(ctx, generation.Table.ParentMenuID); err != nil {
		return nil, err
	}
	return &adminv1.PreviewCodeGenResponse{
		Files:        generation.Files,
		OutputPaths:  generation.OutputPaths,
		MissingI18ns: codegen.MissingI18nFields(table, columns, codegen.LocaleState{Current: localeState.Current, Enabled: localeState.Enabled, Primary: localeState.Primary}),
	}, nil
}

// codeGenLocaleState 查询代码生成使用的数据库语言状态。
func (c *CodeGenCase) codeGenLocaleState(ctx context.Context) (codegen.LocaleState, error) {
	state, err := c.baseLanguageCase.LocaleState(ctx)
	if err != nil {
		return codegen.LocaleState{}, err
	}
	return codegen.LocaleState{Current: state.Current, Enabled: state.Enabled, Primary: state.Primary}, nil
}

// StartCodeGenTask 校验生成对象并创建后台批量任务。
func (c *CodeGenCase) StartCodeGenTask(ctx context.Context, req *adminv1.StartCodeGenTaskRequest) (*adminv1.StartCodeGenTaskResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var batch *codeGenBatchContext
	batch, err = c.prepareCodeGenBatch(ctx, req.GetTableIds())
	if err != nil {
		return nil, err
	}
	tables := make([]*adminv1.CodeGenTaskTable, 0, len(req.GetTableIds()))
	for _, tableID := range req.GetTableIds() {
		generation := batch.plan.GenerationForTable(tableID)
		if generation == nil || generation.Table == nil {
			return nil, errorsx.Internal("批量生成计划缺少表配置")
		}
		tables = append(tables, &adminv1.CodeGenTaskTable{
			TableId:   tableID,
			TableName: generation.Table.TableName_,
			Status:    adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_PENDING,
			Message:   codegen.Message(batch.localeState, "progress.pending_execute", nil),
			Steps:     codegen.BuildProgressSteps(generation.Files, codegen.ShouldSyncMenus(generation.Table, generation.GeneratedMethods), generation.Table.GenBackend == 1, batch.localeState),
		})
	}
	task, created := c.progressManager.Create(authInfo.UserId, tables, batch.localeState)
	if !created {
		return nil, errorsx.StateConflict("已有代码生成任务正在执行", "code_gen_task", "running", "completed")
	}
	go c.runCodeGenTask(context.WithoutCancel(ctx), task.GetTaskId(), req.GetTableIds())
	return &adminv1.StartCodeGenTaskResponse{TaskId: task.GetTaskId()}, nil
}

// RestoreCodeGen 还原单个或批量代码生成结果。
func (c *CodeGenCase) RestoreCodeGen(ctx context.Context, tableIDs []int64) error {
	ids, err := normalizeCodeGenTableIDs(tableIDs)
	if err != nil {
		return err
	}
	manifests := make(map[int64]*codeGenRestoreManifest, len(ids))
	for _, tableID := range ids {
		var manifest *codeGenRestoreManifest
		manifest, err = loadCodeGenRestoreManifest(tableID)
		if err != nil {
			return err
		}
		manifests[tableID] = manifest
	}
	if err = validateCodeGenRestoreBatch(ids, manifests); err != nil {
		return err
	}
	for _, manifest := range manifests {
		if err = validateCodeGenRestoreFiles(manifest.Files); err != nil {
			return err
		}
	}
	paths := make([]string, 0)
	for _, manifest := range manifests {
		for _, file := range manifest.Files {
			paths = append(paths, file.Path)
		}
	}
	for _, tableID := range ids {
		paths = append(paths, codeGenRestoreManifestPath(tableID))
	}
	var fileTransaction *codeGenRestoreTransaction
	fileTransaction, err = newCodeGenRestoreTransaction(paths)
	if err != nil {
		return err
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		for _, tableID := range ids {
			err = c.restoreCodeGenTable(txCtx, tableID, manifests[tableID], fileTransaction)
			if err != nil {
				return err
			}
		}
		for _, tableID := range ids {
			fileTransaction.record(codeGenRestoreManifestPath(tableID))
			var manifestPath string
			manifestPath, err = codegen.SafeRepoFilePath(codeGenRestoreManifestPath(tableID))
			if err != nil {
				return err
			}
			err = os.Remove(manifestPath)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		return nil
	})
	if err != nil {
		rollbackErr := fileTransaction.rollback()
		if rollbackErr != nil {
			return errorsx.Internal("还原代码生成结果失败").WithCause(fmt.Errorf("%w；回滚快照失败：%v", err, rollbackErr))
		}
		return err
	}
	fileTransaction.commit()
	return nil
}

// latestMigrationVersion 查询代码生成使用的最近一次已记录迁移版本。
func (c *CodeGenCase) latestMigrationVersion(ctx context.Context) (string, error) {
	database := c.GormClients[gorm.DefaultClientName]
	return c.baseMigrationCase.LatestVersion(ctx, migration.ModuleName, database.Name())
}

// runCodeGenTask 串行执行批量任务并汇总最终状态。
func (c *CodeGenCase) runCodeGenTask(
	ctx context.Context,
	taskID string,
	tableIDs []int64,
) {
	// 文件、生成产物和格式化都会改写共享工作树，整批任务必须串行执行。
	codeGenGenerationProcessLock.Lock()
	defer codeGenGenerationProcessLock.Unlock()
	c.progressManager.MarkTaskRunning(ctx, taskID)

	batch, err := c.prepareCodeGenBatch(ctx, tableIDs)
	if err != nil {
		c.failCodeGenTask(ctx, taskID, tableIDs, err)
		return
	}
	var beforeSnapshot map[string]codeGenRestoreWorkspaceFile
	beforeSnapshot, err = captureCodeGenWorkspaceSnapshot(batch.plan.Files)
	if err != nil {
		c.failCodeGenTask(ctx, taskID, tableIDs, err)
		return
	}
	beforeMenusByTable := make(map[int64][]*models.BaseMenu, len(tableIDs))
	for _, tableID := range tableIDs {
		generation := batch.plan.GenerationForTable(tableID)
		if generation == nil || !codegen.ShouldSyncMenus(generation.Table, generation.GeneratedMethods) {
			continue
		}
		var menus []*models.BaseMenu
		menus, err = c.listGeneratedMenus(
			ctx,
			generation.Table,
			batch.columnsByTable[tableID],
			generation.GeneratedMethods,
			codegen.FrontendPageComponentPath(generation.OutputPaths.GetFrontendPageFilePath()),
			batch.localeState,
		)
		if err != nil {
			c.failCodeGenTask(ctx, taskID, tableIDs, err)
			return
		}
		beforeMenusByTable[tableID] = cloneBaseMenus(menus)
	}
	reporters := make(map[int64]*codeGenProgressReporter, len(tableIDs))
	for _, tableID := range tableIDs {
		generation := batch.plan.GenerationForTable(tableID)
		if generation == nil || generation.Table == nil {
			c.failCodeGenTask(ctx, taskID, tableIDs, errorsx.Internal("批量生成计划缺少表配置"))
			return
		}
		reporter := &codeGenProgressReporter{manager: c.progressManager, taskID: taskID, tableID: tableID}
		reporters[tableID] = reporter
		c.progressManager.MarkTableRunning(ctx, taskID, tableID)
		c.progressManager.RegisterSteps(ctx, taskID, tableID, codegen.BuildProgressSteps(generation.Files, codegen.ShouldSyncMenus(generation.Table, generation.GeneratedMethods), generation.Table.GenBackend == 1, batch.localeState))
	}
	workflowCtx, cancelWorkflow := context.WithTimeout(context.WithoutCancel(ctx), codegen.WorkflowTimeout)
	defer cancelWorkflow()
	var fileTransaction *codeGenFileTransaction
	generatedMenuIDsByTable := make(map[int64][]int64, len(tableIDs))
	err = c.tx.Transaction(workflowCtx, func(txCtx context.Context) error {
		for _, tableID := range tableIDs {
			generation := batch.plan.GenerationForTable(tableID)
			if !codegen.ShouldSyncMenus(generation.Table, generation.GeneratedMethods) {
				continue
			}
			reporters[tableID].updateStep(txCtx, codegen.MenuStepID, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_RUNNING, codegen.Message(batch.localeState, "progress.running_sync", nil), "")
			err = c.syncGeneratedMenus(txCtx, generation.Table, batch.columnsByTable[tableID], generation.GeneratedMethods, codegen.FrontendPageComponentPath(generation.OutputPaths.GetFrontendPageFilePath()), batch.localeState)
			if err != nil {
				return err
			}
			var menus []*models.BaseMenu
			menus, err = c.listGeneratedMenus(
				txCtx,
				generation.Table,
				batch.columnsByTable[tableID],
				generation.GeneratedMethods,
				codegen.FrontendPageComponentPath(generation.OutputPaths.GetFrontendPageFilePath()),
				batch.localeState,
			)
			if err != nil {
				return err
			}
			generatedMenuIDsByTable[tableID] = baseMenuIDs(menus)
		}
		fileTransaction, err = newCodeGenFileTransaction(batch.plan.Files)
		if err != nil {
			return err
		}
		return c.writeCodeGenBatchFiles(txCtx, batch.plan, reporters, fileTransaction, batch.localeState)
	})
	if err != nil {
		rollbackErr := fileTransaction.rollback()
		if rollbackErr != nil {
			err = errors.Join(err, fmt.Errorf("还原生成内容失败: %w", rollbackErr))
		}
		c.failCodeGenTask(ctx, taskID, tableIDs, err)
		return
	}
	fileTransaction.commit()
	c.completeCodeGenBatchSteps(workflowCtx, batch.plan, reporters, batch.localeState)

	failedCount := 0
	commandTargets := make([]codeGenCommandTarget, 0, len(tableIDs))
	for _, tableID := range tableIDs {
		generation := batch.plan.GenerationForTable(tableID)
		reporter := reporters[tableID]
		if generation.Table.GenBackend == 1 {
			commandTargets = append(commandTargets, codeGenCommandTarget{tableID: tableID, tableName: generation.Table.TableName_, progress: reporter})
			continue
		}
		// 仅在该表全部生成步骤成功后更新持久化状态，失败任务保留原状态便于再次执行。
		err = c.markCodeGenTableGenerated(workflowCtx, tableID)
		if err != nil {
			failedCount++
			c.progressManager.MarkTableCompleted(ctx, taskID, tableID, adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED, codegen.Message(batch.localeState, "progress.generation_state_update_failed", map[string]string{"error": codegen.FailureRemark(err)}))
			continue
		}
		c.progressManager.MarkTableCompleted(ctx, taskID, tableID, adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_SUCCEEDED, codegen.Message(batch.localeState, "progress.generation_complete", nil))
	}
	if len(commandTargets) > 0 {
		commandResults := c.runCodeGenCommands(workflowCtx, commandTargets, batch.localeState)
		for _, target := range commandTargets {
			result := commandResults[target.tableID]
			if result.err != nil {
				failedCount++
				c.progressManager.MarkTableCompleted(ctx, taskID, target.tableID, adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED, result.message)
				continue
			}
			// 共享命令链成功后再更新状态，确保列表不会提前显示为已生成。
			err = c.markCodeGenTableGenerated(workflowCtx, target.tableID)
			if err != nil {
				failedCount++
				c.progressManager.MarkTableCompleted(ctx, taskID, target.tableID, adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED, codegen.Message(batch.localeState, "progress.generation_state_update_failed", map[string]string{"error": codegen.FailureRemark(err)}))
				continue
			}
			c.progressManager.MarkTableCompleted(ctx, taskID, target.tableID, adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_SUCCEEDED, codegen.Message(batch.localeState, "progress.generation_complete", nil))
		}
	}
	var manifests map[int64]*codeGenRestoreManifest
	manifests, err = buildCodeGenRestoreManifests(taskID, tableIDs, batch.plan, beforeSnapshot, beforeMenusByTable, generatedMenuIDsByTable)
	if err != nil {
		c.failCodeGenTask(ctx, taskID, tableIDs, err)
		return
	}
	if err = SaveCodeGenRestoreManifests(manifests); err != nil {
		c.failCodeGenTask(ctx, taskID, tableIDs, err)
		return
	}
	if failedCount > 0 {
		message := codegen.Message(batch.localeState, "progress.batch_complete_partial", map[string]string{
			"success": strconv.Itoa(len(tableIDs) - failedCount),
			"failed":  strconv.Itoa(failedCount),
		})
		c.progressManager.MarkTaskCompleted(ctx, taskID, adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED, message)
		return
	}
	c.progressManager.MarkTaskCompleted(ctx, taskID, adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_SUCCEEDED, codegen.Message(batch.localeState, "progress.batch_complete", map[string]string{"count": strconv.Itoa(len(tableIDs))}))
}

// prepareCodeGenBatch 加载并校验整批生成快照，所有文件冲突均在写入前返回。
func (c *CodeGenCase) prepareCodeGenBatch(ctx context.Context, tableIDs []int64) (*codeGenBatchContext, error) {
	inputs := make([]codegen.BatchGenerationInput, 0, len(tableIDs))
	columnsByTable := make(map[int64][]*codegen.CodeGenColumn, len(tableIDs))
	tableIDSet := make(map[int64]struct{}, len(tableIDs))
	var migrationVersion string
	var err error
	migrationVersion, err = c.latestMigrationVersion(ctx)
	if err != nil {
		return nil, err
	}
	var localeState codegen.LocaleState
	localeState, err = c.codeGenLocaleState(ctx)
	if err != nil {
		return nil, err
	}
	for _, tableID := range tableIDs {
		if tableID <= 0 {
			return nil, errorsx.InvalidArgument("代码生成表配置ID不能为空")
		}
		if _, exists := tableIDSet[tableID]; exists {
			return nil, errorsx.InvalidArgument("代码生成表配置ID不能重复")
		}
		tableIDSet[tableID] = struct{}{}
		var table *codegen.Table
		var columns []*codegen.CodeGenColumn
		var protos []*codegen.Proto
		table, columns, protos, err = c.loadCodeGenContext(ctx, tableID)
		if err != nil {
			return nil, err
		}
		missingI18ns := codegen.MissingI18nFields(table, columns, localeState)
		if len(missingI18ns) > 0 {
			return nil, errorsx.InvalidArgument("正式生成前请补齐翻译配置：" + strings.Join(missingI18ns, "、"))
		}
		// 停用配置只允许查看，不能写入生成文件。
		if table.Status == codegen.StatusDisabled {
			return nil, errorsx.StateConflict("停用的代码生成表配置不能生成", "code_gen_table", "disabled", "draft_or_generated")
		}
		if !codeGenDatabaseTableNamePattern.MatchString(table.TableName_) {
			return nil, errorsx.InvalidArgument("业务表名只能包含小写字母、数字和下划线，且必须以字母开头")
		}
		inputs = append(inputs, codegen.BatchGenerationInput{
			Table:            table,
			Columns:          columns,
			Methods:          protos,
			TableComment:     table.TableComment,
			MigrationVersion: migrationVersion,
			LocaleState:      localeState,
		})
		columnsByTable[tableID] = columns
	}
	var plan *codegen.BatchGeneration
	plan, err = codegen.PrepareBatchGeneration(inputs)
	if err != nil {
		return nil, errorsx.InvalidArgument(codegen.FailureRemark(err)).WithCause(err)
	}
	for _, generation := range plan.Generations {
		if err = c.validateGeneratedBaseAPIs(ctx, generation); err != nil {
			return nil, err
		}
		if err = c.validateGeneratedOptionMethods(ctx, generation.Table, columnsByTable[generation.Table.ID], generation.GeneratedMethods); err != nil {
			return nil, err
		}
		if err = c.validateCodeGenParentMenu(ctx, generation.Table.ParentMenuID); err != nil {
			return nil, err
		}
		for _, file := range generation.Files {
			if strings.Contains(file.GetMessage(), "文件路径不允许") {
				return nil, errorsx.InvalidArgument(file.GetMessage())
			}
		}
	}
	for _, file := range plan.Files {
		if _, err = codegen.SafeRepoFilePath(file.Path); err != nil {
			return nil, err
		}
	}
	return &codeGenBatchContext{plan: plan, columnsByTable: columnsByTable, localeState: localeState}, nil
}

// validateGeneratedBaseAPIs 按 base_api 中的 HTTP 路由校验生成接口冲突。
func (c *CodeGenCase) validateGeneratedBaseAPIs(ctx context.Context, generation *codegen.Generation) error {
	if c.baseAPICase == nil || generation == nil || generation.Table == nil {
		return nil
	}
	baseAPIs, err := c.baseAPICase.List(ctx)
	if err != nil {
		return err
	}
	apisByRoute := make(map[string][]*models.BaseAPI, len(baseAPIs))
	for _, baseAPI := range baseAPIs {
		if baseAPI == nil {
			continue
		}
		key := strings.ToUpper(baseAPI.Method) + "\x00" + baseAPI.Path
		apisByRoute[key] = append(apisByRoute[key], baseAPI)
	}
	for _, method := range generation.GeneratedMethods {
		httpMethod, path, ok := codegen.GeneratedHTTPRoute(generation.Table, method)
		if !ok {
			continue
		}
		existing := apisByRoute[httpMethod+"\x00"+path]
		if len(existing) == 0 {
			continue
		}
		if method.TargetEntityName != generation.Table.EntityName ||
			method.TriggerType == codegen.TriggerFieldOption ||
			method.TriggerType == codegen.TriggerLeftTree {
			continue
		}
		expectedOperation := codegen.GeneratedRPCPath(generation.Table, method)
		for _, baseAPI := range existing {
			if baseAPI.Operation == expectedOperation || method.GenerateWhenMissing != 1 {
				continue
			}
			return errorsx.Conflict(fmt.Sprintf("生成接口%s %s %s与base_api中的%s重复", method.MethodName, httpMethod, path, baseAPI.Operation))
		}
	}
	return nil
}

// writeCodeGenBatchFiles 将批次合并后的每个文件原子写入一次，成功步骤在事务提交后统一完成。
func (c *CodeGenCase) writeCodeGenBatchFiles(ctx context.Context, plan *codegen.BatchGeneration, reporters map[int64]*codeGenProgressReporter, fileTransaction *codeGenFileTransaction, localeState codegen.LocaleState) error {
	for _, file := range plan.Files {
		for _, ref := range file.Refs {
			reporter := reporters[ref.TableID]
			if reporter != nil {
				reporter.updateStep(ctx, codegen.FileStepID(ref.FileIndex), adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_RUNNING, codegen.Message(localeState, "progress.merging_write", nil), "")
			}
		}
		fullPath, err := codegen.SafeRepoFilePath(file.Path)
		if err == nil {
			err = writeGeneratedFile(fullPath, []byte(file.Content), file.Action)
		}
		if err == nil {
			fileTransaction.recordWritten(fullPath)
		}
		for _, ref := range file.Refs {
			reporter := reporters[ref.TableID]
			if reporter == nil {
				continue
			}
			if err != nil {
				reporter.updateStep(ctx, codegen.FileStepID(ref.FileIndex), adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_FAILED, err.Error(), "")
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// completeCodeGenBatchSteps 在菜单和文件事务提交后更新成功步骤。
func (c *CodeGenCase) completeCodeGenBatchSteps(ctx context.Context, plan *codegen.BatchGeneration, reporters map[int64]*codeGenProgressReporter, localeState codegen.LocaleState) {
	for _, generation := range plan.Generations {
		if codegen.ShouldSyncMenus(generation.Table, generation.GeneratedMethods) {
			reporters[generation.Table.ID].updateStep(ctx, codegen.MenuStepID, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SUCCEEDED, codegen.Message(localeState, "progress.sync_complete", nil), "")
		}
	}
	for _, file := range plan.Files {
		for _, ref := range file.Refs {
			reporter := reporters[ref.TableID]
			if reporter != nil {
				reporter.updateStep(ctx, codegen.FileStepID(ref.FileIndex), adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SUCCEEDED, codegen.Message(localeState, "progress.merged_write", nil), "")
			}
		}
	}
}

// failCodeGenTask 将整批预检或写入失败同步到任务和每个表的进度状态。
func (c *CodeGenCase) failCodeGenTask(ctx context.Context, taskID string, tableIDs []int64, err error) {
	message := codegen.FailureRemark(err)
	for _, tableID := range tableIDs {
		c.progressManager.MarkTableCompleted(ctx, taskID, tableID, adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED, message)
	}
	c.progressManager.MarkTaskCompleted(ctx, taskID, adminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED, c.progressManager.Message(taskID, "progress.batch_failed", map[string]string{"error": message}))
}

// markCodeGenTableGenerated 将完整生成成功的代码生成表标记为已生成。
func (c *CodeGenCase) markCodeGenTableGenerated(ctx context.Context, tableID int64) error {
	return c.codeGenTableCase.UpdateByID(ctx, &models.CodeGenTable{ID: tableID, Status: codegen.StatusGenerated})
}

// runCodeGenCommands 对选中业务表执行单表模型生成，并整批执行一次共享生成链。
func (c *CodeGenCase) runCodeGenCommands(ctx context.Context, targets []codeGenCommandTarget, localeState codegen.LocaleState) map[int64]codeGenCommandResult {
	type commandState struct {
		failureMessages []string
		err             error
	}

	backendDir := codegen.BackendDir()
	databaseSource := ""
	dataConfig := c.GetConfig().GetData()
	if dataConfig != nil {
		database := dataConfig.GetDatabase()
		if database == nil {
			database = dataConfig.GetDatabases()[gorm.DefaultClientName]
		}
		if database != nil {
			databaseSource = database.GetSource()
		}
	}
	states := make(map[int64]*commandState, len(targets))
	sharedTargets := []string{"api", "openapi", "ts", "wire"}
	eligibleTargets := make([]codeGenCommandTarget, 0, len(targets))
	for _, target := range targets {
		state := new(commandState)
		states[target.tableID] = state
		stepID := codegen.CommandStepPrefix + "gorm-gen"
		target.progress.updateStep(ctx, stepID, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_RUNNING, codegen.Message(localeState, "progress.running_execute", nil), "")
		if databaseSource == "" {
			state.err = errors.New("代码生成数据库配置缺少数据源")
			failureMessage := codegen.CommandFailureMessage(localeState, "gorm-gen", "", state.err)
			state.failureMessages = append(state.failureMessages, failureMessage)
			target.progress.updateStep(ctx, stepID, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_FAILED, failureMessage, "")
			for _, skippedTarget := range sharedTargets {
				target.progress.updateStep(ctx, codegen.CommandStepPrefix+skippedTarget, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SKIPPED, codegen.Message(localeState, "progress.model_generation_failed", nil), "")
			}
			continue
		}
		output, err := codegen.RunCommand(ctx, backendDir, "gorm-gen", "GORM_TABLE="+target.tableName, "GORM_GEN_SOURCE="+databaseSource)
		if err != nil {
			failureMessage := codegen.CommandFailureMessage(localeState, "gorm-gen", output, err)
			state.failureMessages = append(state.failureMessages, failureMessage)
			state.err = err
			target.progress.updateStep(ctx, stepID, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_FAILED, failureMessage, output)
			for _, skippedTarget := range sharedTargets {
				target.progress.updateStep(ctx, codegen.CommandStepPrefix+skippedTarget, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SKIPPED, codegen.Message(localeState, "progress.model_generation_failed", nil), "")
			}
			continue
		}
		target.progress.updateStep(ctx, stepID, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SUCCEEDED, codegen.Message(localeState, "progress.execute_complete", nil), output)
		eligibleTargets = append(eligibleTargets, target)
	}

	for index, commandTarget := range sharedTargets {
		if len(eligibleTargets) == 0 {
			break
		}
		stepID := codegen.CommandStepPrefix + commandTarget
		for _, target := range eligibleTargets {
			target.progress.updateStep(ctx, stepID, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_RUNNING, codegen.Message(localeState, "progress.running_execute", nil), "")
		}
		output, err := codegen.RunCommand(ctx, backendDir, commandTarget)
		if err != nil {
			failureMessage := codegen.CommandFailureMessage(localeState, commandTarget, output, err)
			for _, target := range eligibleTargets {
				state := states[target.tableID]
				state.failureMessages = append(state.failureMessages, failureMessage)
				state.err = err
				target.progress.updateStep(ctx, stepID, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_FAILED, failureMessage, output)
				for _, skippedTarget := range sharedTargets[index+1:] {
					target.progress.updateStep(ctx, codegen.CommandStepPrefix+skippedTarget, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SKIPPED, codegen.Message(localeState, "progress.previous_command_failed", nil), "")
				}
			}
			break
		}
		for _, target := range eligibleTargets {
			target.progress.updateStep(ctx, stepID, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SUCCEEDED, codegen.Message(localeState, "progress.execute_complete", nil), output)
		}
	}

	formatCtx, cancelFormat := context.WithTimeout(context.WithoutCancel(ctx), codegen.FormatTimeout)
	defer cancelFormat()
	formatStepID := codegen.CommandStepPrefix + "fmt"
	for _, target := range targets {
		target.progress.updateStep(formatCtx, formatStepID, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_RUNNING, codegen.Message(localeState, "progress.running_execute", nil), "")
	}
	fmtOutput, fmtErr := codegen.RunCommand(formatCtx, backendDir, "fmt")
	for _, target := range targets {
		state := states[target.tableID]
		if fmtErr != nil {
			failureMessage := codegen.CommandFailureMessage(localeState, "fmt", fmtOutput, fmtErr)
			state.failureMessages = append(state.failureMessages, failureMessage)
			target.progress.updateStep(formatCtx, formatStepID, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_FAILED, failureMessage, fmtOutput)
			if state.err == nil {
				state.err = fmtErr
			} else {
				state.err = fmt.Errorf("%w；make fmt: %v", state.err, fmtErr)
			}
			continue
		}
		target.progress.updateStep(formatCtx, formatStepID, adminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SUCCEEDED, codegen.Message(localeState, "progress.execute_complete", nil), fmtOutput)
	}

	results := make(map[int64]codeGenCommandResult, len(states))
	for tableID, state := range states {
		result := codeGenCommandResult{err: state.err}
		if state.err != nil {
			result.message = codegen.TruncateText(strings.Join(state.failureMessages, "；"), codegen.RemarkMaxRunes)
		}
		results[tableID] = result
	}
	return results
}

// loadCodeGenContext 只读加载表、字段和Proto生成配置快照。
func (c *CodeGenCase) loadCodeGenContext(ctx context.Context, tableID int64) (*codegen.Table, []*codegen.CodeGenColumn, []*codegen.Proto, error) {
	if tableID <= 0 {
		return nil, nil, nil, errorsx.InvalidArgument("代码生成表配置ID不能为空")
	}
	tableModel, err := c.codeGenTableCase.FindByID(ctx, tableID)
	if err != nil {
		return nil, nil, nil, err
	}
	err = c.codeGenTableCase.ValidateBusinessModule(ctx, tableModel.BusinessModule)
	if err != nil {
		return nil, nil, nil, err
	}
	var table *codegen.Table
	table, err = codeGenTableToSnapshot(tableModel)
	if err != nil {
		return nil, nil, nil, err
	}
	if table.TableComment == "" {
		var tableInfos []dto.CodeGenDatabaseTable
		tableInfos, err = c.codeGenTableCase.listDatabaseTables(ctx, []string{tableModel.Name})
		if err != nil {
			return nil, nil, nil, err
		}
		if len(tableInfos) > 0 {
			table.TableComment = tableInfos[0].TableComment
		}
	}
	table.TableComment = codegen.DefaultString(table.TableComment, table.BusinessName)
	table.BusinessName = codegen.DefaultString(table.TableComment, table.BusinessName)
	var columnConfigs []*adminv1.CodeGenColumn
	columnConfigs, err = c.codeGenColumnCase.listCodeGenColumns(ctx, tableID)
	if err != nil {
		return nil, nil, nil, err
	}
	var databaseColumns []dto.CodeGenDatabaseColumn
	databaseColumns, err = c.codeGenColumnCase.listDatabaseColumns(ctx, tableModel.Name)
	if err != nil {
		return nil, nil, nil, err
	}
	columns := codeGenColumnsToSnapshots(columnConfigs, databaseColumns)
	if err = validateCodeGenStatusDefaults(columns); err != nil {
		return nil, nil, nil, err
	}
	var protoChecks *adminv1.ListCodeGenProtoResponse
	protoChecks, err = c.codeGenProtoCase.ListCodeGenProto(ctx, tableID)
	if err != nil {
		return nil, nil, nil, err
	}
	var protos []*codegen.Proto
	protos, err = codeGenProtosToSnapshots(protoChecks.GetCodeGenProtos(), table.TableComment)
	if err != nil {
		return nil, nil, nil, err
	}
	err = c.enrichCodeGenProtoTargetComments(ctx, table, columns, protos)
	if err != nil {
		return nil, nil, nil, err
	}
	return table, columns, protos, nil
}

// enrichCodeGenProtoTargetComments 补齐左树和外部选项目标表的业务描述及状态元数据。
func (c *CodeGenCase) enrichCodeGenProtoTargetComments(ctx context.Context, table *codegen.Table, columns []*codegen.CodeGenColumn, protos []*codegen.Proto) error {
	leftTreeConfig := codegen.LeftTreeConfigFromTable(table)
	tableNames := make([]string, 0, len(protos))
	seenTableNames := make(map[string]struct{}, len(protos))
	for _, proto := range protos {
		if proto.TargetEntityName == "" || proto.TargetEntityName == table.EntityName {
			continue
		}
		tableName := stringcase.ToSnakeCase(proto.TargetEntityName)
		if proto.TriggerType == codegen.TriggerLeftTree && leftTreeConfig.SourceValue != "" {
			tableName = leftTreeConfig.SourceValue
		}
		if _, exists := seenTableNames[tableName]; exists {
			continue
		}
		seenTableNames[tableName] = struct{}{}
		tableNames = append(tableNames, tableName)
	}
	var tableInfos []dto.CodeGenDatabaseTable
	var err error
	if len(tableNames) > 0 {
		tableInfos, err = c.codeGenTableCase.listDatabaseTables(ctx, tableNames)
		if err != nil {
			return err
		}
	}
	commentsByTableName := make(map[string]string, len(tableInfos))
	for _, tableInfo := range tableInfos {
		commentsByTableName[tableInfo.TableName] = tableInfo.TableComment
	}
	var configuredTables []*models.CodeGenTable
	if len(tableNames) > 0 {
		query := c.codeGenTableCase.Query(ctx).CodeGenTable
		configuredTables, err = c.codeGenTableCase.List(ctx, repository.Where(query.Name.In(tableNames...)))
		if err != nil {
			return err
		}
		for _, configuredTable := range configuredTables {
			if configuredTable.Comment != "" {
				commentsByTableName[configuredTable.Name] = configuredTable.Comment
			}
		}
	}
	configuredTablesByName := make(map[string]*models.CodeGenTable, len(configuredTables))
	for _, configuredTable := range configuredTables {
		configuredTablesByName[configuredTable.Name] = configuredTable
	}
	columnSnapshotsByEntity := map[string][]*codegen.CodeGenColumn{table.EntityName: columns}
	for _, proto := range protos {
		if !codegen.IsOptionProtoMethod(proto) {
			continue
		}
		targetEntity := codegen.DefaultString(proto.TargetEntityName, table.EntityName)
		if _, exists := columnSnapshotsByEntity[targetEntity]; !exists {
			tableName := stringcase.ToSnakeCase(targetEntity)
			if proto.TriggerType == codegen.TriggerLeftTree && leftTreeConfig.SourceValue != "" {
				tableName = leftTreeConfig.SourceValue
			}
			var databaseColumns []dto.CodeGenDatabaseColumn
			databaseColumns, err = c.codeGenColumnCase.listDatabaseColumns(ctx, tableName)
			if err != nil {
				return err
			}
			var targetConfigs []*adminv1.CodeGenColumn
			if configuredTable := configuredTablesByName[tableName]; configuredTable != nil {
				targetConfigs, err = c.codeGenColumnCase.listCodeGenColumns(ctx, configuredTable.ID)
				if err != nil {
					return err
				}
			}
			columnSnapshotsByEntity[targetEntity] = codeGenColumnsToSnapshots(targetConfigs, databaseColumns)
		}
		statusColumn, statusEnabledValue, ok := codegen.OptionStatusMetadata(columnSnapshotsByEntity[targetEntity])
		if ok {
			proto.OptionStatusColumn = statusColumn
			proto.OptionStatusEnabledValue = statusEnabledValue
		}
	}
	for _, proto := range protos {
		if proto.TargetEntityName == "" || proto.TargetEntityName == table.EntityName {
			proto.TargetBusinessName = table.TableComment
			continue
		}
		tableName := stringcase.ToSnakeCase(proto.TargetEntityName)
		if proto.TriggerType == codegen.TriggerLeftTree && leftTreeConfig.SourceValue != "" {
			tableName = leftTreeConfig.SourceValue
		}
		comment := commentsByTableName[tableName]
		if proto.TriggerType == codegen.TriggerLeftTree && leftTreeConfig.Comment != "" {
			comment = leftTreeConfig.Comment
		}
		proto.TargetBusinessName = codegen.DefaultString(comment, proto.TargetEntityName)
	}
	return nil
}

// validateGeneratedOptionMethods 校验选项接口字段能安全写入公共响应类型。
func (c *CodeGenCase) validateGeneratedOptionMethods(ctx context.Context, table *codegen.Table, columns []*codegen.CodeGenColumn, methods []*codegen.Proto) error {
	targetColumnCache := map[string][]*codegen.CodeGenColumn{table.EntityName: columns}
	for _, method := range methods {
		if !codegen.IsOptionProtoMethod(method) {
			continue
		}
		targetColumns, err := c.optionTargetColumns(ctx, table, method, targetColumnCache)
		if err != nil {
			return err
		}
		if err = validateOptionLabelColumn(method, targetColumns, codegen.DefaultString(method.LabelColumn, "name"), "显示字段"); err != nil {
			return err
		}
		if err = validateOptionIntegerColumn(method, targetColumns, codegen.DefaultString(method.ValueColumn, "id"), "值字段"); err != nil {
			return err
		}
		if method.APIKind == codegen.APIKindTree {
			if err = validateOptionIntegerColumn(method, targetColumns, codegen.DefaultString(method.ParentColumn, "parent_id"), "父节点字段"); err != nil {
				return err
			}
		}
	}
	return nil
}

// optionTargetColumns 查询选项接口目标实体的数据库字段。
func (c *CodeGenCase) optionTargetColumns(ctx context.Context, table *codegen.Table, method *codegen.Proto, cache map[string][]*codegen.CodeGenColumn) ([]*codegen.CodeGenColumn, error) {
	target := codegen.DefaultString(method.TargetEntityName, table.EntityName)
	if target == table.EntityName {
		return cache[table.EntityName], nil
	}
	if columns, exists := cache[target]; exists {
		return columns, nil
	}
	tableName := stringcase.ToSnakeCase(target)
	leftTreeConfig := codegen.LeftTreeConfigFromTable(table)
	if method.TriggerType == codegen.TriggerLeftTree && leftTreeConfig.SourceValue != "" {
		tableName = leftTreeConfig.SourceValue
	}
	databaseColumns, err := c.codeGenColumnCase.listDatabaseColumns(ctx, tableName)
	if err != nil {
		return nil, err
	}
	if len(databaseColumns) == 0 {
		return nil, errorsx.InvalidArgument(fmt.Sprintf("选项接口%s的目标表%s不存在，无法校验字段类型", method.MethodName, tableName))
	}
	columns := codeGenColumnsToSnapshots(nil, databaseColumns)
	cache[target] = columns
	return columns, nil
}

// validateCodeGenParentMenu 校验生成页面的父级菜单节点。
func (c *CodeGenCase) validateCodeGenParentMenu(ctx context.Context, parentMenuID int64) error {
	if parentMenuID <= 0 {
		return errorsx.InvalidArgument("代码生成必须选择固定一级菜单下的父级菜单")
	}
	menu, err := c.baseMenuCase.FindByID(ctx, parentMenuID)
	if err != nil {
		return errorsx.InvalidArgument("父级菜单不存在").WithCause(err)
	}
	return validateBaseMenuChild(menu, _const.BASE_MENU_TYPE_MENU)
}

// syncGeneratedMenus 在当前事务中幂等同步生成页面及按钮权限菜单。
func (c *CodeGenCase) syncGeneratedMenus(ctx context.Context, table *codegen.Table, columns []*codegen.CodeGenColumn, methods []*codegen.Proto, resourcePath string, localeState codegen.LocaleState) error {
	pageSpec, buttonSpecs := codegen.MenuSpecs(table, columns, methods, resourcePath, table.TableComment, localeState)
	pageMenu, err := c.upsertGeneratedPageMenu(ctx, pageSpec)
	if err != nil {
		return err
	}
	if err = c.baseMenuCase.SaveGeneratedMenuI18ns(ctx, pageMenu.ID, pageSpec.SourceTitle, pageSpec.I18ns); err != nil {
		return err
	}
	for _, buttonSpec := range buttonSpecs {
		buttonSpec.Menu.ParentID = pageMenu.ID
		var buttonMenu *models.BaseMenu
		buttonMenu, err = c.upsertGeneratedButtonMenu(ctx, buttonSpec)
		if err != nil {
			return err
		}
		if err = c.baseMenuCase.SaveGeneratedMenuI18ns(ctx, buttonMenu.ID, buttonSpec.SourceTitle, buttonSpec.I18ns); err != nil {
			return err
		}
	}
	return c.disableStaleGeneratedStatusMenus(ctx, pageMenu.ID, table, buttonSpecs)
}

// disableStaleGeneratedStatusMenus 停用本轮不再需要的状态按钮权限。
func (c *CodeGenCase) disableStaleGeneratedStatusMenus(ctx context.Context, pageMenuID int64, table *codegen.Table, buttonSpecs []codegen.CodeGenMenuSpec) error {
	expectedPaths := make(map[string]struct{}, len(buttonSpecs))
	for _, buttonSpec := range buttonSpecs {
		expectedPaths[buttonSpec.Menu.Path] = struct{}{}
	}
	query := c.baseMenuCase.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ParentID.Eq(pageMenuID)))
	opts = append(opts, repository.Where(query.Type.Eq(_const.BASE_MENU_TYPE_BUTTON)))
	menus, err := c.baseMenuCase.List(ctx, opts...)
	if err != nil {
		return err
	}
	statusPathPrefix := codegen.PermissionPrefix(table) + ":status"
	statusAPIPrefix := codegen.GeneratedRPCServicePath(table, table.EntityName) + "/Set"
	for _, menu := range menus {
		if _, exists := expectedPaths[menu.Path]; exists {
			continue
		}
		if menu.Path != statusPathPrefix && !strings.HasPrefix(menu.Path, statusPathPrefix+":") && !strings.Contains(menu.API, statusAPIPrefix) {
			continue
		}
		if menu.Status == coreconst.STATUS_STATUS_DISABLE && menu.API == "[]" {
			continue
		}
		if err = c.baseMenuCase.Update(
			ctx,
			&models.BaseMenu{ID: menu.ID, Status: coreconst.STATUS_STATUS_DISABLE, API: "[]"},
			repository.Where(query.ID.Eq(menu.ID)),
			repository.Select(query.Status, query.API),
		); err != nil {
			return err
		}
		if err = c.baseMenuCase.casbinRuleCase.RebuildCasbinRuleByMenuID(ctx, menu.ID); err != nil {
			return err
		}
	}
	return nil
}

// upsertGeneratedPageMenu 创建或更新生成页面菜单。
func (c *CodeGenCase) upsertGeneratedPageMenu(ctx context.Context, spec codegen.CodeGenMenuSpec) (*models.BaseMenu, error) {
	query := c.baseMenuCase.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(
		query.Type.Eq(_const.BASE_MENU_TYPE_MENU),
		field.Or(
			query.Path.Eq(spec.Menu.Path),
			query.Name.Eq(spec.Menu.Name),
			query.Component.Eq(spec.Menu.Component),
		),
	))
	menus, err := c.baseMenuCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	if len(menus) == 0 {
		if err = c.baseMenuCase.createBaseMenu(ctx, spec.Menu); err != nil {
			return nil, err
		}
		return spec.Menu, nil
	}
	if menus[0].ParentID != spec.Menu.ParentID {
		return nil, errorsx.StateConflict("已生成菜单不能更换父级，请先删除原菜单后重新生成", "base_menu", fmt.Sprint(menus[0].ParentID), fmt.Sprint(spec.Menu.ParentID))
	}
	spec.Menu.ID = menus[0].ID
	if err = c.baseMenuCase.UpdateByID(ctx, spec.Menu); err != nil {
		return nil, err
	}
	if err = c.baseMenuCase.casbinRuleCase.RebuildCasbinRuleByMenuID(ctx, spec.Menu.ID); err != nil {
		return nil, err
	}
	return spec.Menu, nil
}

// upsertGeneratedButtonMenu 创建或更新生成按钮权限菜单。
func (c *CodeGenCase) upsertGeneratedButtonMenu(ctx context.Context, spec codegen.CodeGenMenuSpec) (*models.BaseMenu, error) {
	query := c.baseMenuCase.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ParentID.Eq(spec.Menu.ParentID)))
	opts = append(opts, repository.Where(query.Type.Eq(_const.BASE_MENU_TYPE_BUTTON)))
	menus, err := c.baseMenuCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	for _, menu := range menus {
		if menu.Path != spec.Menu.Path && menu.API != spec.Menu.API {
			continue
		}
		spec.Menu.ID = menu.ID
		if err = c.baseMenuCase.UpdateByID(ctx, spec.Menu); err != nil {
			return nil, err
		}
		if err = c.baseMenuCase.casbinRuleCase.RebuildCasbinRuleByMenuID(ctx, spec.Menu.ID); err != nil {
			return nil, err
		}
		return spec.Menu, nil
	}
	if err = c.baseMenuCase.createBaseMenu(ctx, spec.Menu); err != nil {
		return nil, err
	}
	return spec.Menu, nil
}

// updateStep 更新当前生成对象的单个执行步骤。
func (r *codeGenProgressReporter) updateStep(ctx context.Context, stepID string, status adminv1.CodeGenTaskStepStatus, message string, output string) {
	if r == nil {
		return
	}
	r.manager.UpdateStep(ctx, r.taskID, r.tableID, stepID, status, message, output)
}

// codeGenTableToSnapshot 将现有表配置转换为生成器只读快照。
func codeGenTableToSnapshot(item *models.CodeGenTable) (*codegen.Table, error) {
	target, ok := codegen.ProtoTargetForBusinessModule(item.BusinessModule)
	if !ok {
		return nil, errorsx.InvalidArgument("业务模块不能为空或格式不正确")
	}
	pathSegments := strings.Split(item.Name, "_")
	modulePath := item.Name
	if len(pathSegments) > 1 {
		modulePath = strings.Join(pathSegments[:len(pathSegments)-1], "/")
	}
	entityName := stringcase.ToPascalCase(item.Name)
	businessName := codegen.DefaultString(item.Comment, item.Name)
	leftTreeConfig := ""
	var err error
	if item.LeftTreeConfig != "" {
		var config adminv1.CodeGenLeftTreeConfig
		err = json.Unmarshal([]byte(item.LeftTreeConfig), &config)
		if err != nil {
			return nil, errorsx.Internal("左树配置格式错误").WithCause(err)
		}
		var data []byte
		data, err = json.Marshal(codegen.CodeGenLeftTreeConfig{
			Enabled:      item.PageType == codegen.PageTypeLeftTree,
			SourceType:   codegen.OptionSourceTable,
			SourceValue:  config.GetTableName(),
			Comment:      config.GetComment(),
			FilterColumn: config.GetFilterColumn(),
			ParentColumn: config.GetParentColumn(),
			LabelColumn:  config.GetLabelColumn(),
			ValueColumn:  config.GetValueColumn(),
			Lazy:         config.GetLazy(),
		})
		if err != nil {
			return nil, errorsx.Internal("转换左树配置失败").WithCause(err)
		}
		leftTreeConfig = string(data)
	}
	i18nConfig := make(map[string]codegen.LocaleConfig)
	if item.I18NConfig != "" {
		var config map[string]*adminv1.CodeGenLocaleConfig
		err = json.Unmarshal([]byte(item.I18NConfig), &config)
		if err != nil {
			return nil, errorsx.Internal("表级国际化配置格式错误").WithCause(err)
		}
		i18nConfig = codeGenLocaleConfigsToSnapshots(config)
	}
	return &codegen.Table{
		ID:               item.ID,
		TableName_:       item.Name,
		TableComment:     item.Comment,
		BusinessModule:   item.BusinessModule,
		BusinessName:     businessName,
		EntityName:       entityName,
		ModulePath:       modulePath,
		APIPath:          target.Directory,
		PermissionPrefix: strings.Join(pathSegments, ":"),
		ParentMenuID:     item.ParentMenuID,
		PageType:         item.PageType,
		ParentColumn:     item.ParentColumn,
		TreeLabelColumn:  item.TreeLabelColumn,
		LeftTreeConfig:   leftTreeConfig,
		I18NConfig:       i18nConfig,
		GenBackend:       item.GenBackend,
		GenFrontend:      item.GenFrontend,
		GenSql:           item.GenSql,
		Status:           item.Status,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}, nil
}

// codeGenColumnsToSnapshots 将完整字段配置与数据库元数据合并为生成快照。
func codeGenColumnsToSnapshots(configs []*adminv1.CodeGenColumn, databaseColumns []dto.CodeGenDatabaseColumn) []*codegen.CodeGenColumn {
	configByName := make(map[string]*adminv1.CodeGenColumn, len(configs))
	for _, config := range configs {
		configByName[config.GetName()] = config
	}
	columns := make([]*codegen.CodeGenColumn, 0, len(databaseColumns))
	for index, databaseColumn := range databaseColumns {
		config := configByName[databaseColumn.Name]
		if config == nil {
			config = newDefaultCodeGenColumn(0, databaseColumn, int32(index+1), codeGenColumnDefaultSource{})
		}
		queryConfig := config.GetQueryConfig()
		listConfig := config.GetListConfig()
		formConfig := config.GetFormConfig()
		queryOption := codeGenOptionToSnapshot(queryConfig.GetOption())
		listOption := codeGenOptionToSnapshot(listConfig.GetOption())
		formOption := codeGenOptionToSnapshot(formConfig.GetOption())
		isStatus := listConfig.GetEnabled() && listConfig.GetComponent() == "switch"
		defaultValue := ""
		hasDefault := databaseColumn.ColumnDefault.Valid
		if hasDefault {
			defaultValue = databaseColumn.ColumnDefault.String
		}
		statusEnabledValue := codegen.DefaultString(listOption.ActiveValue, "1")
		statusDisabledValue := codegen.DefaultString(listOption.InactiveValue, "2")
		column := &codegen.CodeGenColumn{
			ID:                  config.GetId(),
			TableID:             config.GetTableId(),
			Name:                databaseColumn.Name,
			Comment:             config.GetComment(),
			I18NConfig:          codeGenLocaleConfigsToSnapshots(config.GetI18NConfig()),
			DbType:              databaseColumn.DataType,
			ColumnType:          databaseColumn.ColumnType,
			DbLength:            config.GetDbLength(),
			DbScale:             config.GetDbScale(),
			DefaultValue:        defaultValue,
			HasDefault:          hasDefault,
			Extra:               databaseColumn.Extra,
			IsPrimary:           codegen.BoolToInt32(config.GetIsPrimary()),
			IsAutoIncrement:     codegen.BoolToInt32(config.GetIsAutoIncrement()),
			IsNullable:          codegen.BoolToInt32(config.GetIsNullable()),
			GoType:              config.GetGoType(),
			ProtoType:           config.GetProtoType(),
			TsType:              config.GetTsType(),
			IsQuery:             codegen.BoolToInt32(queryConfig.GetEnabled()),
			QueryOperator:       codegen.NormalizeQueryOperator(queryConfig.GetOperator()),
			QueryComponent:      queryConfig.GetComponent(),
			IsList:              codegen.BoolToInt32(listConfig.GetEnabled()),
			ListComponent:       listConfig.GetComponent(),
			IsForm:              codegen.BoolToInt32(formConfig.GetEnabled()),
			FormComponent:       formConfig.GetComponent(),
			IsRequired:          codegen.BoolToInt32(formConfig.GetRequired()),
			FormMultiple:        formConfig.GetMultiple(),
			QueryOption:         queryOption,
			ListOption:          listOption,
			FormOption:          formOption,
			IsStatusField:       codegen.BoolToInt32(isStatus),
			StatusDataType:      listOption.SourceType,
			StatusDictCode:      listOption.SourceValue,
			StatusEnabledValue:  statusEnabledValue,
			StatusDisabledValue: statusDisabledValue,
			StatusDefaultValue:  defaultValue,
			StatusGenerateAPI:   codegen.BoolToInt32(isStatus),
			StatusTableColumn:   codegen.BoolToInt32(isStatus),
			StatusSearch:        codegen.BoolToInt32(isStatus && queryConfig.GetEnabled()),
			StatusSwitch:        codegen.BoolToInt32(isStatus),
			StatusForm:          codegen.BoolToInt32(isStatus && formConfig.GetEnabled()),
			Sort:                config.GetSort(),
		}
		columns = append(columns, column)
	}
	return columns
}

// codeGenLocaleConfigsToSnapshots 将协议国际化配置转换为生成器只读快照。
func codeGenLocaleConfigsToSnapshots(configs map[string]*adminv1.CodeGenLocaleConfig) map[string]codegen.LocaleConfig {
	result := make(map[string]codegen.LocaleConfig, len(configs))
	for localeValue, config := range configs {
		if config == nil {
			continue
		}
		result[localeValue] = codegen.LocaleConfig{
			Comment:         config.GetComment(),
			LeftTreeComment: config.GetLeftTreeComment(),
		}
	}
	return result
}

// validateCodeGenStatusDefaults 校验状态字段数据库默认值属于启用或禁用值。
func validateCodeGenStatusDefaults(columns []*codegen.CodeGenColumn) error {
	for _, column := range columns {
		if column == nil || column.IsStatusField != 1 || column.StatusDefaultValue == "" {
			continue
		}
		if column.StatusDefaultValue == column.StatusEnabledValue || column.StatusDefaultValue == column.StatusDisabledValue {
			continue
		}
		return errorsx.InvalidArgument(fmt.Sprintf("字段%s的状态默认值%s不属于启用值%s或禁用值%s", column.Name, column.StatusDefaultValue, column.StatusEnabledValue, column.StatusDisabledValue))
	}
	return nil
}

// codeGenOptionToSnapshot 转换单个作用域的字段选项配置。
func codeGenOptionToSnapshot(option *adminv1.CodeGenColumnOptionConfig) codegen.CodeGenColumnOptionConfig {
	return codegen.CodeGenColumnOptionConfig{
		Kind:          option.GetKind(),
		SourceType:    option.GetSourceType(),
		SourceValue:   option.GetSourceValue(),
		LabelField:    option.GetLabelField(),
		ValueField:    option.GetValueField(),
		ParentField:   option.GetParentField(),
		ActiveValue:   option.GetActiveValue(),
		InactiveValue: option.GetInactiveValue(),
		Lazy:          option.GetLazy(),
	}
}

// codeGenProtosToSnapshots 将现有Proto配置转换为生成器只读快照。
func codeGenProtosToSnapshots(items []*adminv1.CodeGenProtoCheck, tableComment string) ([]*codegen.Proto, error) {
	protos := make([]*codegen.Proto, 0, len(items))
	for _, item := range items {
		config := item.GetConfig()
		columnName := ""
		if item.GetApiKind() == codegen.APIKindStatus {
			columnName = config.GetStatusColumn()
		}
		protos = append(protos, &codegen.Proto{
			TableID:            item.GetTableId(),
			Name:               columnName,
			TriggerType:        item.GetTriggerType(),
			APIKind:            item.GetApiKind(),
			TargetEntityName:   item.GetTargetEntityName(),
			TargetBusinessName: codegen.DefaultString(tableComment, item.GetTargetEntityName()),
			MethodName:         item.GetMethodName(),
			ProtoFilePath:      item.GetProtoFilePath(),
			ParentColumn:       config.GetParentColumn(),
			LabelColumn:        config.GetLabelColumn(),
			ValueColumn:        config.GetValueColumn(),
			Lazy:               config.GetLazy(),
			// 已存在的接口也要参与候选渲染，才能同步表单字段等消息结构变化。
			GenerateWhenMissing: codegen.BoolToInt32(item.GetGenerateWhenMissing() || item.GetExists()),
			Sort:                item.GetSort(),
		})
	}
	return protos, nil
}

// validateOptionLabelColumn 校验选项响应的显示字段存在。
func validateOptionLabelColumn(method *codegen.Proto, columns []*codegen.CodeGenColumn, columnName string, fieldLabel string) error {
	if codegen.FindColumnByName(columns, columnName) == nil {
		return errorsx.InvalidArgument(fmt.Sprintf("选项接口%s的%s%s不存在", method.MethodName, fieldLabel, columnName))
	}
	return nil
}

// validateOptionIntegerColumn 校验选项响应字段可转换为int64。
func validateOptionIntegerColumn(method *codegen.Proto, columns []*codegen.CodeGenColumn, columnName string, fieldLabel string) error {
	column := codegen.FindColumnByName(columns, columnName)
	if column == nil {
		return errorsx.InvalidArgument(fmt.Sprintf("选项接口%s的%s%s不存在", method.MethodName, fieldLabel, columnName))
	}
	goType := codegen.DefaultString(column.GoType, codegen.InferGoType(column.DbType))
	if goType != "int64" && goType != "int32" {
		return errorsx.InvalidArgument(fmt.Sprintf("选项接口%s的%s%s必须是整数类型字段", method.MethodName, fieldLabel, columnName))
	}
	return nil
}

// newCodeGenFileTransaction 在写入前保存本批目标文件快照。
func newCodeGenFileTransaction(files []*codegen.BatchFile) (*codeGenFileTransaction, error) {
	transaction := &codeGenFileTransaction{snapshots: make(map[string]codeGenFileSnapshot, len(files))}
	for _, file := range files {
		fullPath, err := codegen.SafeRepoFilePath(file.Path)
		if err != nil {
			return nil, err
		}
		if _, exists := transaction.snapshots[fullPath]; exists {
			continue
		}
		var fileInfo os.FileInfo
		fileInfo, err = os.Stat(fullPath)
		if os.IsNotExist(err) {
			transaction.snapshots[fullPath] = codeGenFileSnapshot{fullPath: fullPath}
			continue
		}
		if err != nil {
			return nil, err
		}
		var content []byte
		content, err = os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		transaction.snapshots[fullPath] = codeGenFileSnapshot{
			fullPath: fullPath,
			exists:   true,
			content:  content,
			mode:     fileInfo.Mode().Perm(),
		}
	}
	return transaction, nil
}

// recordWritten 记录已被当前生成事务成功写入的文件。
func (t *codeGenFileTransaction) recordWritten(fullPath string) {
	if t == nil || t.committed {
		return
	}
	t.writtenPaths = append(t.writtenPaths, fullPath)
}

// commit 提交文件生成事务，使已写入内容保留在工作区。
func (t *codeGenFileTransaction) commit() {
	if t == nil {
		return
	}
	t.committed = true
	t.writtenPaths = nil
}

// rollback 恢复本次生成已经写入的文件，并删除本次新建的文件。
func (t *codeGenFileTransaction) rollback() error {
	if t == nil || t.committed {
		return nil
	}
	var rollbackErr error
	for index := len(t.writtenPaths) - 1; index >= 0; index-- {
		snapshot, exists := t.snapshots[t.writtenPaths[index]]
		if !exists {
			continue
		}
		var err error
		if snapshot.exists {
			err = writeGeneratedFileAtomically(snapshot.fullPath, snapshot.content, snapshot.mode, false)
		} else {
			err = os.Remove(snapshot.fullPath)
			if os.IsNotExist(err) {
				err = nil
			}
		}
		if err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
	}
	return rollbackErr
}

// writeGeneratedFile 通过同目录临时文件原子写入生成内容。
func writeGeneratedFile(fullPath string, content []byte, action string) error {
	mode := os.FileMode(0o644)
	var fileInfo os.FileInfo
	fileInfo, err := os.Stat(fullPath)
	// 创建动作禁止覆盖已有文件，更新动作保留原文件权限。
	switch action {
	case "create":
		if err == nil {
			return fmt.Errorf("生成文件已存在: %s", fullPath)
		}
		if !os.IsNotExist(err) {
			return err
		}
	case "update":
		if err != nil {
			return err
		}
		mode = fileInfo.Mode().Perm()
	default:
		return fmt.Errorf("不支持的生成动作: %s", action)
	}
	return writeGeneratedFileAtomically(fullPath, content, mode, action == "create")
}

// writeGeneratedFileAtomically 将内容写入临时文件后替换目标文件。
func writeGeneratedFileAtomically(fullPath string, content []byte, mode os.FileMode, createOnly bool) error {
	err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
	if err != nil {
		return err
	}
	var tempFile *os.File
	tempFile, err = os.CreateTemp(filepath.Dir(fullPath), "."+filepath.Base(fullPath)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()
	if err = tempFile.Chmod(mode); err != nil {
		_ = tempFile.Close()
		return err
	}
	if _, err = tempFile.Write(content); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err = tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err = tempFile.Close(); err != nil {
		return err
	}
	if createOnly {
		return os.Link(tempPath, fullPath)
	}
	return os.Rename(tempPath, fullPath)
}

// RestoreAvailable 判断代码生成表是否存在可用还原快照。
func RestoreAvailable(tableID int64) bool {
	if tableID <= 0 {
		return false
	}
	path, err := codegen.SafeRepoFilePath(codeGenRestoreManifestPath(tableID))
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// captureCodeGenWorkspaceSnapshot 读取生成命令可能改写的工作区文件。
func captureCodeGenWorkspaceSnapshot(batchFiles []*codegen.BatchFile) (map[string]codeGenRestoreWorkspaceFile, error) {
	paths, err := codeGenWorkspacePaths()
	if err != nil {
		return nil, err
	}
	for _, file := range batchFiles {
		if _, exists := paths[file.Path]; !exists {
			paths[file.Path] = struct{}{}
		}
	}
	snapshots := make(map[string]codeGenRestoreWorkspaceFile, len(paths))
	for path := range paths {
		var snapshot codeGenRestoreWorkspaceFile
		snapshot, err = readCodeGenWorkspaceFile(path)
		if err != nil {
			return nil, err
		}
		snapshots[path] = snapshot
	}
	return snapshots, nil
}

// buildCodeGenRestoreManifests 构建本次生成实际改写文件的还原快照。
func buildCodeGenRestoreManifests(
	taskID string,
	tableIDs []int64,
	batch *codegen.BatchGeneration,
	before map[string]codeGenRestoreWorkspaceFile,
	menusByTable map[int64][]*models.BaseMenu,
	generatedMenuIDsByTable map[int64][]int64,
) (map[int64]*codeGenRestoreManifest, error) {
	after, err := captureCodeGenWorkspaceSnapshot(batch.Files)
	if err != nil {
		return nil, err
	}
	ownersByPath := make(map[string][]int64)
	for _, file := range batch.Files {
		owners := make([]int64, 0, len(file.Refs))
		for _, ref := range file.Refs {
			owners = appendUniqueInt64(owners, ref.TableID)
		}
		ownersByPath[file.Path] = owners
	}
	manifests := make(map[int64]*codeGenRestoreManifest, len(tableIDs))
	for _, tableID := range tableIDs {
		manifests[tableID] = &codeGenRestoreManifest{
			Version:          2,
			TaskID:           taskID,
			TableID:          tableID,
			BatchTableIDs:    append([]int64(nil), tableIDs...),
			Menus:            cloneBaseMenus(menusByTable[tableID]),
			GeneratedMenuIDs: append([]int64(nil), generatedMenuIDsByTable[tableID]...),
		}
	}
	allPaths := make(map[string]struct{}, len(before)+len(after))
	for path := range before {
		allPaths[path] = struct{}{}
	}
	for path := range after {
		allPaths[path] = struct{}{}
	}
	for path := range allPaths {
		afterFile, afterExists := after[path]
		if !afterExists {
			afterFile = codeGenRestoreWorkspaceFile{Path: path}
		}
		beforeFile, beforeExists := before[path]
		if !beforeExists {
			beforeFile = codeGenRestoreWorkspaceFile{Path: path}
		}
		if workspaceFilesEqual(beforeFile, afterFile) {
			continue
		}
		owners := append([]int64(nil), ownersByPath[path]...)
		if len(owners) == 0 {
			owners = append(owners, tableIDs...)
		}
		for _, tableID := range owners {
			manifest := manifests[tableID]
			if manifest == nil {
				continue
			}
			manifest.Files = append(manifest.Files, codeGenRestoreFile{
				Path:             path,
				OriginalExists:   beforeFile.Exists,
				OriginalContent:  string(beforeFile.Content),
				OriginalMode:     uint32(beforeFile.Mode.Perm()),
				GeneratedExists:  afterFile.Exists,
				GeneratedContent: string(afterFile.Content),
				OwnerTableIDs:    append([]int64(nil), owners...),
			})
		}
	}
	for _, manifest := range manifests {
		sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	}
	return manifests, nil
}

// cloneBaseMenus 复制菜单快照，避免事务中的更新改写还原数据。
func cloneBaseMenus(menus []*models.BaseMenu) []*models.BaseMenu {
	clones := make([]*models.BaseMenu, 0, len(menus))
	for _, menu := range menus {
		if menu == nil {
			continue
		}
		clone := *menu
		clones = append(clones, &clone)
	}
	return clones
}

// baseMenuIDs 提取菜单快照中的稳定主键。
func baseMenuIDs(menus []*models.BaseMenu) []int64 {
	ids := make([]int64, 0, len(menus))
	for _, menu := range menus {
		if menu != nil && menu.ID > 0 {
			ids = appendUniqueInt64(ids, menu.ID)
		}
	}
	return ids
}

// SaveCodeGenRestoreManifests 持久化本次生成的还原快照。
func SaveCodeGenRestoreManifests(manifests map[int64]*codeGenRestoreManifest) error {
	paths := make([]string, 0, len(manifests))
	for tableID := range manifests {
		paths = append(paths, codeGenRestoreManifestPath(tableID))
	}
	transaction, err := newCodeGenRestoreTransaction(paths)
	if err != nil {
		return err
	}
	for tableID, manifest := range manifests {
		var content []byte
		content, err = json.Marshal(manifest)
		if err != nil {
			break
		}
		var manifestPath string
		manifestPath, err = codegen.SafeRepoFilePath(codeGenRestoreManifestPath(tableID))
		if err != nil {
			break
		}
		err = writeCodeGenRestoreFile(manifestPath, content)
		if err != nil {
			break
		}
		transaction.record(codeGenRestoreManifestPath(tableID))
	}
	if err != nil {
		rollbackErr := transaction.rollback()
		if rollbackErr != nil {
			return errorsx.Internal("保存代码生成还原快照失败").WithCause(fmt.Errorf("%w；回滚快照失败：%v", err, rollbackErr))
		}
		return err
	}
	transaction.commit()
	return nil
}

// listGeneratedMenus 查询当前代码生成对象关联的页面和按钮菜单。
func (c *CodeGenCase) listGeneratedMenus(ctx context.Context, table *codegen.Table, columns []*codegen.CodeGenColumn, methods []*codegen.Proto, resourcePath string, localeState codegen.LocaleState) ([]*models.BaseMenu, error) {
	pageSpec, buttonSpecs := codegen.MenuSpecs(table, columns, methods, resourcePath, table.TableComment, localeState)
	query := c.baseMenuCase.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.Type.Eq(_const.BASE_MENU_TYPE_MENU)))
	opts = append(opts, repository.Where(field.Or(
		query.Path.Eq(pageSpec.Menu.Path),
		query.Name.Eq(pageSpec.Menu.Name),
		query.Component.Eq(pageSpec.Menu.Component),
	)))
	pages, err := c.baseMenuCase.List(ctx, opts...)
	if err != nil || len(pages) == 0 {
		return pages, err
	}
	page := pages[0]
	menus := []*models.BaseMenu{page}
	childOpts := make([]repository.QueryOption, 0, 2)
	childOpts = append(childOpts, repository.Where(query.ParentID.Eq(page.ID)))
	childOpts = append(childOpts, repository.Where(query.Type.Eq(_const.BASE_MENU_TYPE_BUTTON)))
	var children []*models.BaseMenu
	children, err = c.baseMenuCase.List(ctx, childOpts...)
	if err != nil {
		return nil, err
	}
	expectedPaths := make(map[string]struct{}, len(buttonSpecs))
	expectedAPIs := make(map[string]struct{}, len(buttonSpecs))
	for _, spec := range buttonSpecs {
		expectedPaths[spec.Menu.Path] = struct{}{}
		expectedAPIs[spec.Menu.API] = struct{}{}
	}
	statusPathPrefix := codegen.PermissionPrefix(table) + ":status"
	statusAPIPrefix := codegen.GeneratedRPCServicePath(table, table.EntityName) + "/Set"
	for _, child := range children {
		_, expectedPath := expectedPaths[child.Path]
		_, expectedAPI := expectedAPIs[child.API]
		if expectedPath || expectedAPI || strings.HasPrefix(child.Path, statusPathPrefix) || strings.Contains(child.API, statusAPIPrefix) {
			menus = append(menus, child)
		}
	}
	return menus, nil
}

// restoreCodeGenTable 还原单个代码生成对象的文件、菜单和状态。
func (c *CodeGenCase) restoreCodeGenTable(ctx context.Context, tableID int64, manifest *codeGenRestoreManifest, transaction *codeGenRestoreTransaction) error {
	table, columns, protos, err := c.loadCodeGenContext(ctx, tableID)
	if err != nil {
		return err
	}
	var localeState codegen.LocaleState
	localeState, err = c.codeGenLocaleState(ctx)
	if err != nil {
		return err
	}
	var generation *codegen.Generation
	generation, err = codegen.PrepareGeneration(table, columns, protos, nil, table.TableComment, localeState)
	if err != nil {
		return err
	}
	if manifest.Version == 1 && codegen.ShouldSyncMenus(generation.Table, generation.GeneratedMethods) {
		err = c.removeGeneratedMenus(ctx, generation.Table, columns, generation.GeneratedMethods, codegen.FrontendPageComponentPath(generation.OutputPaths.GetFrontendPageFilePath()), localeState)
		if err != nil {
			return err
		}
	}
	if manifest.Version >= 2 {
		if err = c.restoreGeneratedMenus(ctx, manifest); err != nil {
			return err
		}
	}
	for _, file := range manifest.Files {
		var fullPath string
		fullPath, err = codegen.SafeRepoFilePath(file.Path)
		if err != nil {
			return err
		}
		if file.OriginalExists {
			mode := os.FileMode(file.OriginalMode)
			if mode == 0 {
				mode = 0o644
			}
			err = writeGeneratedFileAtomically(fullPath, []byte(file.OriginalContent), mode, false)
		} else {
			err = os.Remove(fullPath)
			if os.IsNotExist(err) {
				err = nil
			}
		}
		if err != nil {
			return err
		}
		transaction.record(file.Path)
	}
	query := c.codeGenTableCase.Query(ctx).CodeGenTable
	return c.codeGenTableCase.Update(
		ctx,
		&models.CodeGenTable{ID: tableID, Status: codegen.StatusDraft},
		repository.Where(query.ID.Eq(tableID)),
		repository.Select(query.Status),
	)
}

// restoreGeneratedMenus 将菜单数据库恢复到代码生成前状态。
func (c *CodeGenCase) restoreGeneratedMenus(ctx context.Context, manifest *codeGenRestoreManifest) error {
	if manifest == nil || len(manifest.GeneratedMenuIDs) == 0 {
		return nil
	}
	currentMenus, err := c.baseMenuCase.ListByIDs(ctx, manifest.GeneratedMenuIDs)
	if err != nil {
		return err
	}
	beforeByID := make(map[int64]*models.BaseMenu, len(manifest.Menus))
	for _, menu := range manifest.Menus {
		if menu == nil {
			continue
		}
		beforeByID[menu.ID] = menu
	}
	currentByID := make(map[int64]*models.BaseMenu, len(currentMenus))
	for _, menu := range currentMenus {
		if menu != nil {
			currentByID[menu.ID] = menu
		}
	}
	deleteIDs := make([]int64, 0, len(currentMenus))
	for _, menu := range currentMenus {
		if menu == nil {
			continue
		}
		if _, exists := beforeByID[menu.ID]; !exists {
			deleteIDs = append(deleteIDs, menu.ID)
		}
	}
	if len(deleteIDs) > 0 {
		if err = c.baseMenuCase.DeleteByIDs(ctx, deleteIDs); err != nil {
			return err
		}
	}
	query := c.baseMenuCase.Query(ctx).BaseMenu
	for _, menu := range manifest.Menus {
		if menu == nil {
			continue
		}
		if _, exists := currentByID[menu.ID]; !exists {
			if err = c.baseMenuCase.Create(ctx, menu); err != nil {
				return err
			}
		} else {
			err = c.baseMenuCase.Update(
				ctx,
				menu,
				repository.Where(query.ID.Eq(menu.ID)),
				repository.Select(
					query.ParentID,
					query.Type,
					query.Path,
					query.Name,
					query.Component,
					query.Redirect,
					query.Meta,
					query.API,
					query.Sort,
					query.Status,
					query.CreatedBy,
					query.UpdatedBy,
					query.CreatedAt,
					query.UpdatedAt,
					query.DeletedAt,
				),
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// removeGeneratedMenus 删除代码生成对象产生的页面和按钮权限。
func (c *CodeGenCase) removeGeneratedMenus(ctx context.Context, table *codegen.Table, columns []*codegen.CodeGenColumn, methods []*codegen.Proto, resourcePath string, localeState codegen.LocaleState) error {
	menus, err := c.listGeneratedMenus(ctx, table, columns, methods, resourcePath, localeState)
	if err != nil || len(menus) == 0 {
		return err
	}
	ids := make([]int64, 0, len(menus)-1)
	for index := len(menus) - 1; index >= 1; index-- {
		ids = append(ids, menus[index].ID)
	}
	if len(ids) > 0 {
		err = c.baseMenuCase.DeleteByIDs(ctx, ids)
		if err != nil {
			return err
		}
	}
	query := c.baseMenuCase.Query(ctx).BaseMenu
	var childCount int64
	childCount, err = c.baseMenuCase.Count(ctx, repository.Where(query.ParentID.Eq(menus[0].ID)))
	if err != nil {
		return err
	}
	if childCount > 0 {
		err = c.baseMenuCase.Update(
			ctx,
			&models.BaseMenu{ID: menus[0].ID, Status: coreconst.STATUS_STATUS_DISABLE, API: "[]"},
			repository.Where(query.ID.Eq(menus[0].ID)),
			repository.Select(query.Status, query.API),
		)
		if err != nil {
			return err
		}
		return nil
	}
	err = c.baseMenuCase.DeleteByIDs(ctx, []int64{menus[0].ID})
	if err != nil {
		return err
	}
	return nil
}

// validateCodeGenRestoreFiles 确认工作区仍是生成后的快照，避免覆盖人工修改。
func validateCodeGenRestoreFiles(files []codeGenRestoreFile) error {
	for _, file := range files {
		fullPath, err := codegen.SafeRepoFilePath(file.Path)
		if err != nil {
			return err
		}
		var current []byte
		current, err = os.ReadFile(fullPath)
		currentExists := err == nil
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if currentExists != file.GeneratedExists || currentExists && string(current) != file.GeneratedContent {
			return errorsx.StateConflict("生成文件已被修改，无法安全还原", "code_gen_file", file.Path, "generated")
		}
	}
	return nil
}

// validateCodeGenRestoreBatch 校验批量生成共享文件必须整批还原。
func validateCodeGenRestoreBatch(ids []int64, manifests map[int64]*codeGenRestoreManifest) error {
	selected := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		selected[id] = struct{}{}
	}
	for _, manifest := range manifests {
		for _, batchID := range manifest.BatchTableIDs {
			if _, ok := selected[batchID]; !ok {
				return errorsx.StateConflict("批量生成的共享文件必须整批还原", "code_gen_task", manifest.TaskID, "restore")
			}
		}
	}
	return nil
}

// normalizeCodeGenTableIDs 去重并校验代码生成表配置 ID。
func normalizeCodeGenTableIDs(tableIDs []int64) ([]int64, error) {
	ids := make([]int64, 0, len(tableIDs))
	seen := make(map[int64]struct{}, len(tableIDs))
	for _, tableID := range tableIDs {
		if tableID <= 0 {
			return nil, errorsx.InvalidArgument("代码生成表配置ID必须大于0")
		}
		if _, ok := seen[tableID]; ok {
			continue
		}
		seen[tableID] = struct{}{}
		ids = append(ids, tableID)
	}
	if len(ids) == 0 {
		return nil, errorsx.InvalidArgument("请选择还原的代码生成表")
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

// codeGenRestoreManifestPath 返回代码生成还原快照路径。
func codeGenRestoreManifestPath(tableID int64) string {
	return filepath.ToSlash(filepath.Join(codeGenRestoreRoot, fmt.Sprintf("%d.json", tableID)))
}

// loadCodeGenRestoreManifest 加载单个代码生成还原快照。
func loadCodeGenRestoreManifest(tableID int64) (*codeGenRestoreManifest, error) {
	path, err := codegen.SafeRepoFilePath(codeGenRestoreManifestPath(tableID))
	if err != nil {
		return nil, err
	}
	var content []byte
	content, err = os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, errorsx.StateConflict("代码生成还原快照不存在，请重新生成", "code_gen_table", fmt.Sprint(tableID), "restore")
	}
	if err != nil {
		return nil, err
	}
	manifest := new(codeGenRestoreManifest)
	if err = json.Unmarshal(content, manifest); err != nil {
		return nil, errorsx.Internal("代码生成还原快照格式错误").WithCause(err)
	}
	if manifest.TableID != tableID || (manifest.Version != 1 && manifest.Version != 2) {
		return nil, errorsx.StateConflict("代码生成还原快照已失效，请重新生成", "code_gen_table", fmt.Sprint(tableID), "restore")
	}
	return manifest, nil
}

// newCodeGenRestoreTransaction 创建还原快照文件事务。
func newCodeGenRestoreTransaction(paths []string) (*codeGenRestoreTransaction, error) {
	transaction := &codeGenRestoreTransaction{snapshots: make(map[string]codeGenFileSnapshot, len(paths))}
	for _, path := range paths {
		fullPath, err := codegen.SafeRepoFilePath(path)
		if err != nil {
			return nil, err
		}
		if _, exists := transaction.snapshots[fullPath]; exists {
			continue
		}
		var fileInfo os.FileInfo
		fileInfo, err = os.Stat(fullPath)
		if os.IsNotExist(err) {
			transaction.snapshots[fullPath] = codeGenFileSnapshot{fullPath: fullPath}
			continue
		}
		if err != nil {
			return nil, err
		}
		var content []byte
		content, err = os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		transaction.snapshots[fullPath] = codeGenFileSnapshot{fullPath: fullPath, exists: true, content: content, mode: fileInfo.Mode().Perm()}
	}
	return transaction, nil
}

// record 记录还原快照文件写入。
func (t *codeGenRestoreTransaction) record(path string) {
	if t == nil || t.committed {
		return
	}
	fullPath, err := codegen.SafeRepoFilePath(path)
	if err == nil {
		t.writtenPaths = append(t.writtenPaths, fullPath)
	}
}

// rollback 回滚还原快照文件变更。
func (t *codeGenRestoreTransaction) rollback() error {
	if t == nil || t.committed {
		return nil
	}
	var rollbackErr error
	for index := len(t.writtenPaths) - 1; index >= 0; index-- {
		snapshot := t.snapshots[t.writtenPaths[index]]
		var err error
		if snapshot.exists {
			err = writeGeneratedFileAtomically(snapshot.fullPath, snapshot.content, snapshot.mode, false)
		} else {
			err = os.Remove(snapshot.fullPath)
			if os.IsNotExist(err) {
				err = nil
			}
		}
		if err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
	}
	return rollbackErr
}

// commit 提交还原快照文件事务。
func (t *codeGenRestoreTransaction) commit() {
	if t != nil {
		t.committed = true
		t.writtenPaths = nil
	}
}

// writeCodeGenRestoreFile 原子写入还原快照文件。
func writeCodeGenRestoreFile(path string, content []byte) error {
	return writeGeneratedFileAtomically(path, content, 0o644, false)
}

// codeGenWorkspacePaths 收集生成命令可能改写的源代码路径。
func codeGenWorkspacePaths() (map[string]struct{}, error) {
	paths := make(map[string]struct{})
	backendPath, err := codegen.SafeRepoFilePath("backend")
	if err != nil {
		return nil, err
	}
	rootPath := filepath.Dir(backendPath)
	roots := []string{
		filepath.Join(rootPath, "backend"),
		filepath.Join(rootPath, "frontend/admin/packages/core/src/rpc"),
		filepath.Join(rootPath, "frontend/admin/packages/modules"),
	}
	for _, root := range roots {
		if _, err = os.Stat(root); os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if path == filepath.Join(rootPath, "backend/data") || path == filepath.Join(rootPath, "backend/logs") {
					return filepath.SkipDir
				}
				return nil
			}
			if !isCodeGenWorkspaceFile(path) {
				return nil
			}
			var relative string
			relative, err = filepath.Rel(rootPath, path)
			if err != nil {
				return err
			}
			paths[filepath.ToSlash(relative)] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return paths, nil
}

// isCodeGenWorkspaceFile 判断文件是否属于生成命令可能改写的源代码产物。
func isCodeGenWorkspaceFile(path string) bool {
	normalizedPath := "/" + strings.TrimPrefix(filepath.ToSlash(path), "/")
	switch filepath.Ext(path) {
	case ".go", ".proto", ".ts", ".vue":
		return true
	case ".json":
		return strings.Contains(normalizedPath, "/frontend/admin/packages/modules/") && strings.Contains(normalizedPath, "/src/locales/")
	default:
		return false
	}
}

// readCodeGenWorkspaceFile 读取仓库相对路径文件状态。
func readCodeGenWorkspaceFile(path string) (codeGenRestoreWorkspaceFile, error) {
	fullPath, err := codegen.SafeRepoFilePath(path)
	if err != nil {
		return codeGenRestoreWorkspaceFile{}, err
	}
	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(fullPath)
	if os.IsNotExist(err) {
		return codeGenRestoreWorkspaceFile{Path: path}, nil
	}
	if err != nil {
		return codeGenRestoreWorkspaceFile{}, err
	}
	var content []byte
	content, err = os.ReadFile(fullPath)
	if err != nil {
		return codeGenRestoreWorkspaceFile{}, err
	}
	return codeGenRestoreWorkspaceFile{Path: path, Exists: true, Content: content, Mode: fileInfo.Mode().Perm()}, nil
}

// workspaceFilesEqual 判断两个工作区文件快照是否一致。
func workspaceFilesEqual(left codeGenRestoreWorkspaceFile, right codeGenRestoreWorkspaceFile) bool {
	return left.Exists == right.Exists && (!left.Exists || reflect.DeepEqual(left.Content, right.Content))
}

// appendUniqueInt64 向 ID 切片追加未出现的值。
func appendUniqueInt64(values []int64, value int64) []int64 {
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}
