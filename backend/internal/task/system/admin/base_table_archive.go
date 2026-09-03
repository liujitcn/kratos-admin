package admin

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/backup"
	adminbiz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	coreBiz "github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/oss"
	"github.com/liujitcn/kratos-kit/transport/cron"
)

const (
	// TableArchiveTaskName 是表归档任务的稳定调用目标。
	TableArchiveTaskName = "system.admin.BaseTableArchive"
	archiveBatchSize     = 5000
)

var (
	archiveTableNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
)

var _ cron.TaskExec = (*TableArchiveTask)(nil)

// TableArchiveTask 按归档配置将当前数据源中的历史表数据保存到内部归档表或 OSS。
type TableArchiveTask struct {
	baseCase    *coreBiz.BaseCase
	archiveRepo *data.BaseTableArchiveRepository
	recordRepo  *data.BaseTableArchiveRecordRepository
}

// NewTableArchiveTask 创建表归档任务。
func NewTableArchiveTask(baseCase *coreBiz.BaseCase, archiveRepo *data.BaseTableArchiveRepository, recordRepo *data.BaseTableArchiveRecordRepository) *TableArchiveTask {
	return &TableArchiveTask{baseCase: baseCase, archiveRepo: archiveRepo, recordRepo: recordRepo}
}

// Task 返回交由 base_job 调度的任务定义。
func (t *TableArchiveTask) Task() cron.Task {
	return cron.Task{Name: TableArchiveTaskName, Exec: t}
}

// Exec 执行所有启用的表归档配置。
func (t *TableArchiveTask) Exec(ctx context.Context, _ map[string]string) ([]string, error) {
	query := t.archiveRepo.Query(ctx).BaseTableArchive
	configs, err := t.archiveRepo.List(ctx, repository.Where(query.Status.Eq(coreconst.STATUS_STATUS_ENABLE)), repository.Order(query.ID.Asc()))
	if err != nil {
		return nil, fmt.Errorf("查询表归档配置失败: %w", err)
	}
	if len(configs) == 0 {
		return []string{"没有启用的表归档配置"}, nil
	}
	var archivedRows int64
	var deletedRows int64
	for _, config := range configs {
		var archived int64
		var deleted int64
		archived, deleted, err = t.archiveOne(ctx, config)
		if err != nil {
			return nil, err
		}
		if err = t.cleanupArchiveRetention(ctx, config); err != nil {
			return nil, err
		}
		archivedRows += archived
		deletedRows += deleted
	}
	return []string{fmt.Sprintf("表归档完成：归档 %d 条，删除在线数据 %d 条", archivedRows, deletedRows)}, nil
}

// archiveOne 执行单条表归档配置并记录执行结果。
func (t *TableArchiveTask) archiveOne(ctx context.Context, config *models.BaseTableArchive) (int64, int64, error) {
	resource, err := archiveResource(config.TableName_)
	if err != nil {
		return 0, 0, err
	}
	var client *gorm.Client
	client, err = adminbiz.GormClientBySourceName(t.baseCase, config.SourceName)
	if err != nil {
		return 0, 0, err
	}
	var databaseConfig *configv1.Data_Database
	databaseConfig, err = databaseConfigByName(t.baseCase, config.SourceName)
	if err != nil {
		return 0, 0, err
	}
	var dsn *mysql.Config
	dsn, err = mysql.ParseDSN(databaseConfig.GetSource())
	if err != nil {
		return 0, 0, fmt.Errorf("解析数据源 %s 失败: %w", config.SourceName, err)
	}
	cutoff := time.Now().AddDate(0, 0, -int(config.OnlineRetentionDays))
	record := &models.BaseTableArchiveRecord{
		ArchiveID: config.ID, SourceName: config.SourceName, TableName_: config.TableName_, ArchiveMode: config.ArchiveMode, CutoffAt: cutoff,
		Cursor: "", ArchiveTableName: resource.archiveTableName, ObjectKey: "", SizeBytes: 0, Sha256: "",
		Status: int32(adminv1.BaseTableArchiveRecordStatus_BASE_TABLE_ARCHIVE_RECORD_STATUS_RUNNING), Error: "",
		StartedAt: time.Now(), FinishedAt: time.Now(),
	}
	err = t.recordRepo.Create(ctx, record)
	if err != nil {
		return 0, 0, fmt.Errorf("创建表归档记录失败: %w", err)
	}
	var archivedRows int64
	var deletedRows int64
	if config.ArchiveMode == int32(adminv1.BaseTableArchiveMode_BASE_TABLE_ARCHIVE_MODE_INTERNAL_DATABASE) {
		archivedRows, deletedRows, err = archiveIntoCurrentDatabase(ctx, client, resource, cutoff, config.BatchSize, config.DeleteAfterVerify != 0)
	} else if config.ArchiveMode == int32(adminv1.BaseTableArchiveMode_BASE_TABLE_ARCHIVE_MODE_OSS) {
		archivedRows, deletedRows, record.ObjectKey, record.SizeBytes, record.Sha256, err = archiveIntoOSS(ctx, t.baseCase.OSS, client, dsn, resource, config, cutoff, config.DeleteAfterVerify != 0)
	} else {
		err = fmt.Errorf("表 %s 的归档模式无效", config.TableName_)
	}
	if err != nil {
		record.Status = int32(adminv1.BaseTableArchiveRecordStatus_BASE_TABLE_ARCHIVE_RECORD_STATUS_FAILED)
		record.Error = err.Error()
		record.FinishedAt = time.Now()
		_ = t.recordRepo.UpdateByID(ctx, record)
		return 0, 0, fmt.Errorf("归档表 %s 失败: %w", config.TableName_, err)
	}
	record.ScannedRows = archivedRows
	record.ArchivedRows = archivedRows
	record.DeletedRows = deletedRows
	record.Status = int32(adminv1.BaseTableArchiveRecordStatus_BASE_TABLE_ARCHIVE_RECORD_STATUS_SUCCESS)
	record.FinishedAt = time.Now()
	err = t.recordRepo.UpdateByID(ctx, record)
	if err != nil {
		return archivedRows, deletedRows, fmt.Errorf("更新表归档记录失败: %w", err)
	}
	return archivedRows, deletedRows, nil
}

type archiveResourceDefinition struct {
	tableName        string
	timeColumn       string
	archiveTableName string
}

func archiveResource(tableName string) (archiveResourceDefinition, error) {
	if !archiveTableNamePattern.MatchString(tableName) || strings.HasPrefix(tableName, "base_table_") {
		return archiveResourceDefinition{}, fmt.Errorf("数据表名称不在可归档范围: %s", tableName)
	}
	timeColumn := "created_at"
	if strings.HasSuffix(tableName, "_log") {
		timeColumn = "occurred_at"
	}
	return archiveResourceDefinition{tableName: tableName, timeColumn: timeColumn, archiveTableName: "base_table_archive_" + tableName}, nil
}

// archiveIntoCurrentDatabase 将过期数据复制到当前数据源内部的归档表。
func archiveIntoCurrentDatabase(ctx context.Context, client *gorm.Client, resource archiveResourceDefinition, cutoff time.Time, batchSize int32, deleteAfterVerify bool) (int64, int64, error) {
	if batchSize <= 0 {
		batchSize = archiveBatchSize
	}
	quotedSource := "`" + resource.tableName + "`"
	quotedArchive := "`" + resource.archiveTableName + "`"
	quotedTime := "`" + resource.timeColumn + "`"
	//nolint:forbidigo // 表名和字段名来自受控归档资源定义，时间值通过参数绑定。
	err := client.DB.WithContext(ctx).Exec("CREATE TABLE IF NOT EXISTS " + quotedArchive + " LIKE " + quotedSource).Error
	if err != nil {
		return 0, 0, fmt.Errorf("创建内部归档表失败: %w", err)
	}
	if !deleteAfterVerify {
		//nolint:forbidigo // 表名和字段名来自受控归档资源定义，时间值通过参数绑定。
		result := client.DB.WithContext(ctx).Exec("INSERT IGNORE INTO "+quotedArchive+" SELECT * FROM "+quotedSource+" WHERE "+quotedTime+" < ?", cutoff)
		if result.Error != nil {
			return 0, 0, fmt.Errorf("复制内部归档数据失败: %w", result.Error)
		}
		return result.RowsAffected, 0, nil
	}
	var archivedRows int64
	var deletedRows int64
	for {
		ids, listErr := listArchiveIDs(ctx, client, resource, cutoff, batchSize)
		if listErr != nil {
			return archivedRows, deletedRows, listErr
		}
		if len(ids) == 0 {
			break
		}
		idList := archiveIDList(ids)
		//nolint:forbidigo // 表名来自受控归档资源定义，主键来自同一数据源的查询结果。
		result := client.DB.WithContext(ctx).Exec("INSERT IGNORE INTO " + quotedArchive + " SELECT * FROM " + quotedSource + " WHERE `id` IN (" + idList + ")")
		if result.Error != nil {
			return archivedRows, deletedRows, fmt.Errorf("复制内部归档数据失败: %w", result.Error)
		}
		verifiedRows, verifyErr := countArchiveIDs(ctx, client, resource.archiveTableName, ids)
		if verifyErr != nil {
			return archivedRows, deletedRows, verifyErr
		}
		if verifiedRows != int64(len(ids)) {
			return archivedRows, deletedRows, fmt.Errorf("内部归档数据校验数量不一致")
		}
		//nolint:forbidigo // 表名和字段名来自受控归档资源定义，主键来自同一数据源的查询结果。
		deleteResult := client.DB.WithContext(ctx).Exec("DELETE FROM "+quotedSource+" WHERE `id` IN ("+idList+") AND "+quotedTime+" < ?", cutoff)
		if deleteResult.Error != nil {
			return archivedRows, deletedRows, fmt.Errorf("删除在线归档数据失败: %w", deleteResult.Error)
		}
		archivedRows += int64(len(ids))
		deletedRows += deleteResult.RowsAffected
	}
	return archivedRows, deletedRows, nil
}

// archiveIntoOSS 将过期数据导出、压缩并上传到 OSS。
func archiveIntoOSS(ctx context.Context, storage oss.OSS, client *gorm.Client, dsn *mysql.Config, resource archiveResourceDefinition, config *models.BaseTableArchive, cutoff time.Time, deleteAfterVerify bool) (int64, int64, string, int64, string, error) {
	if storage == nil {
		return 0, 0, "", 0, "", fmt.Errorf("OSS 未配置")
	}
	var err error
	var ids []int64
	var count int64
	var where string
	if deleteAfterVerify {
		batchSize := config.BatchSize
		if batchSize <= 0 {
			batchSize = archiveBatchSize
		}
		ids, err = listArchiveIDs(ctx, client, resource, cutoff, batchSize)
		if err != nil {
			return 0, 0, "", 0, "", err
		}
		if len(ids) == 0 {
			return 0, 0, "", 0, "", nil
		}
		count = int64(len(ids))
		where = "id IN (" + archiveIDList(ids) + ")"
	} else {
		count, err = countArchiveRows(ctx, client, resource, cutoff)
		if err != nil || count == 0 {
			return count, 0, "", 0, "", err
		}
		where = resource.timeColumn + " < '" + cutoff.Format("2006-01-02 15:04:05") + "'"
	}
	temporary, err := os.CreateTemp("", "kratos-table-archive-*.sql")
	if err != nil {
		return 0, 0, "", 0, "", fmt.Errorf("创建表归档临时文件失败: %w", err)
	}
	temporaryPath := temporary.Name()
	if err = temporary.Close(); err != nil {
		return 0, 0, "", 0, "", fmt.Errorf("关闭表归档临时文件失败: %w", err)
	}
	defer os.Remove(temporaryPath)
	useCommand := backup.CommandAvailable(backup.MysqldumpCommand)
	var sqlDB *sql.DB
	if !useCommand {
		sqlDB, err = client.DB.DB()
		if err != nil {
			return 0, 0, "", 0, "", fmt.Errorf("获取数据源 SQL 连接失败: %w", err)
		}
	}
	args := []string{"--single-transaction", "--no-create-info", "--skip-triggers"}
	if dsn.Net == "unix" && dsn.Addr != "" {
		args = append(args, "--socket="+dsn.Addr)
	} else {
		host := dsn.Addr
		port := "3306"
		parsedHost, parsedPort, splitErr := net.SplitHostPort(dsn.Addr)
		if splitErr == nil {
			host = parsedHost
			port = parsedPort
		}
		args = append(args, "--host="+host, "--port="+port)
	}
	if dsn.User != "" {
		args = append(args, "--user="+dsn.User)
	}
	args = append(args, dsn.DBName, resource.tableName, "--where="+where, "--result-file="+temporaryPath)
	if useCommand {
		var passwordFile string
		passwordFile, err = backup.WriteMySQLDefaultsFile(dsn.Passwd)
		if err != nil {
			return 0, 0, "", 0, "", err
		}
		defer os.Remove(passwordFile)
		args = append([]string{"--defaults-extra-file=" + passwordFile}, args...)
		command := exec.CommandContext(ctx, backup.MysqldumpCommand, args...)
		if err = command.Run(); err != nil {
			return 0, 0, "", 0, "", fmt.Errorf("导出表归档数据失败: %w", err)
		}
	} else {
		var file *os.File
		file, err = os.OpenFile(temporaryPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return 0, 0, "", 0, "", fmt.Errorf("创建 Go 表归档文件失败: %w", err)
		}
		dumpErr := backup.DumpMySQL(ctx, sqlDB, backup.MySQLDumpOptions{
			Database:    dsn.DBName,
			Table:       resource.tableName,
			Where:       where,
			IncludeData: true,
		}, file)
		if dumpErr == nil {
			dumpErr = file.Sync()
		}
		closeErr := file.Close()
		if dumpErr != nil {
			return 0, 0, "", 0, "", fmt.Errorf("Go 导出表归档数据失败: %w", dumpErr)
		}
		if closeErr != nil {
			return 0, 0, "", 0, "", fmt.Errorf("关闭 Go 表归档文件失败: %w", closeErr)
		}
	}
	dataValue, err := os.ReadFile(temporaryPath)
	if err != nil {
		return 0, 0, "", 0, "", fmt.Errorf("读取表归档数据失败: %w", err)
	}
	digest := sha256.Sum256(dataValue)
	prefix := strings.Trim(config.OSSPrefix, "/")
	objectKey := fmt.Sprintf("%s/%s/%s/%s.sql", prefix, config.SourceName, resource.tableName, time.Now().UTC().Format("20060102-150405"))
	_, err = storage.UploadByByte(resource.tableName+".sql", objectKey, dataValue)
	if err != nil {
		return 0, 0, "", 0, "", fmt.Errorf("上传表归档对象失败: %w", err)
	}
	verifiedValue, err := storage.GetFileByte(objectKey)
	if err != nil {
		_ = storage.DeleteFile(objectKey)
		return 0, 0, objectKey, int64(len(dataValue)), hex.EncodeToString(digest[:]), fmt.Errorf("回读表归档对象失败: %w", err)
	}
	verifiedDigest := sha256.Sum256(verifiedValue)
	if !hmac.Equal(digest[:], verifiedDigest[:]) {
		_ = storage.DeleteFile(objectKey)
		return 0, 0, objectKey, int64(len(dataValue)), hex.EncodeToString(digest[:]), fmt.Errorf("表归档对象 SHA-256 校验失败")
	}
	var deletedRows int64
	if deleteAfterVerify {
		quotedSource := "`" + resource.tableName + "`"
		quotedTime := "`" + resource.timeColumn + "`"
		//nolint:forbidigo // 表名和字段名来自受控归档资源定义，主键来自同一数据源的查询结果。
		deleteResult := client.DB.WithContext(ctx).Exec("DELETE FROM "+quotedSource+" WHERE `id` IN ("+archiveIDList(ids)+") AND "+quotedTime+" < ?", cutoff)
		if deleteResult.Error != nil {
			return count, 0, objectKey, int64(len(dataValue)), hex.EncodeToString(digest[:]), fmt.Errorf("删除在线归档数据失败: %w", deleteResult.Error)
		}
		deletedRows = deleteResult.RowsAffected
	}
	return count, deletedRows, objectKey, int64(len(dataValue)), hex.EncodeToString(digest[:]), nil
}

func countArchiveRows(ctx context.Context, client *gorm.Client, resource archiveResourceDefinition, cutoff time.Time) (int64, error) {
	var count int64
	//nolint:forbidigo // 表名和字段名来自受控归档资源定义，时间值通过参数绑定。
	row := client.DB.WithContext(ctx).Raw("SELECT COUNT(*) FROM `"+resource.tableName+"` WHERE `"+resource.timeColumn+"` < ?", cutoff).Row()
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("统计表归档数据失败: %w", err)
	}
	return count, nil
}

// listArchiveIDs 查询本次归档候选记录的主键，删除阶段只使用该集合。
func listArchiveIDs(ctx context.Context, client *gorm.Client, resource archiveResourceDefinition, cutoff time.Time, batchSize int32) ([]int64, error) {
	//nolint:forbidigo // 表名和字段名来自受控归档资源定义，时间值和批量大小通过参数绑定。
	rows, err := client.DB.WithContext(ctx).Raw("SELECT `id` FROM `"+resource.tableName+"` WHERE `"+resource.timeColumn+"` < ? ORDER BY `id` ASC LIMIT ?", cutoff, batchSize).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询待归档记录主键失败: %w", err)
	}
	defer rows.Close()
	ids := make([]int64, 0, batchSize)
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("读取待归档记录主键失败: %w", err)
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历待归档记录主键失败: %w", err)
	}
	return ids, nil
}

// countArchiveIDs 校验归档表中已存在本次候选主键。
func countArchiveIDs(ctx context.Context, client *gorm.Client, archiveTableName string, ids []int64) (int64, error) {
	var count int64
	//nolint:forbidigo // 表名和主键来自受控归档资源定义及同一数据源查询结果。
	row := client.DB.WithContext(ctx).Raw("SELECT COUNT(*) FROM `" + archiveTableName + "` WHERE `id` IN (" + archiveIDList(ids) + ")").Row()
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("校验归档记录失败: %w", err)
	}
	return count, nil
}

// archiveIDList 将已查询的主键集合编码为安全的 SQL 数字列表。
func archiveIDList(ids []int64) string {
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, strconv.FormatInt(id, 10))
	}
	return strings.Join(values, ",")
}

// cleanupArchiveRetention 清理超过归档保留期的内部归档数据或 OSS 对象。
func (t *TableArchiveTask) cleanupArchiveRetention(ctx context.Context, config *models.BaseTableArchive) error {
	if config.ArchiveRetentionDays <= 0 {
		return nil
	}
	query := t.recordRepo.Query(ctx).BaseTableArchiveRecord
	cutoff := time.Now().AddDate(0, 0, -int(config.ArchiveRetentionDays))
	records, err := t.recordRepo.List(ctx,
		repository.Where(query.ArchiveID.Eq(config.ID)),
		repository.Where(query.Status.Eq(int32(adminv1.BaseTableArchiveRecordStatus_BASE_TABLE_ARCHIVE_RECORD_STATUS_SUCCESS))),
		repository.Where(query.FinishedAt.Lt(cutoff)),
		repository.Order(query.ID.Asc()),
	)
	if err != nil {
		return fmt.Errorf("查询过期归档记录失败: %w", err)
	}
	for _, record := range records {
		if record.ObjectKey != "" {
			if t.baseCase.OSS == nil {
				return fmt.Errorf("清理归档对象失败: OSS 未配置")
			}
			if err = t.baseCase.OSS.DeleteFile(record.ObjectKey); err != nil {
				return fmt.Errorf("删除过期归档对象失败: %w", err)
			}
		}
		if record.ArchiveTableName != "" && config.ArchiveMode == int32(adminv1.BaseTableArchiveMode_BASE_TABLE_ARCHIVE_MODE_INTERNAL_DATABASE) {
			var client *gorm.Client
			client, err = adminbiz.GormClientBySourceName(t.baseCase, record.SourceName)
			if err != nil {
				return fmt.Errorf("清理内部归档表失败: %w", err)
			}
			var resource archiveResourceDefinition
			resource, err = archiveResource(record.TableName_)
			if err != nil {
				return err
			}
			if !archiveTableNamePattern.MatchString(record.ArchiveTableName) {
				return fmt.Errorf("内部归档表名不合法: %s", record.ArchiveTableName)
			}
			quotedArchive := "`" + record.ArchiveTableName + "`"
			quotedTime := "`" + resource.timeColumn + "`"
			//nolint:forbidigo // 表名和字段名来自受控归档资源定义，时间值通过参数绑定。
			if result := client.DB.WithContext(ctx).Exec("DELETE FROM "+quotedArchive+" WHERE "+quotedTime+" < ?", cutoff); result.Error != nil {
				return fmt.Errorf("删除内部归档数据失败: %w", result.Error)
			}
		}
		record.Status = int32(adminv1.BaseTableArchiveRecordStatus_BASE_TABLE_ARCHIVE_RECORD_STATUS_DELETED)
		if err = t.recordRepo.UpdateByID(ctx, record); err != nil {
			return fmt.Errorf("更新过期归档记录失败: %w", err)
		}
	}
	return nil
}

// deleteExpiredRows 按批次删除超过保留期的日志编号，并返回删除数量。
func deleteExpiredRows(ctx context.Context, fetch func(context.Context) ([]int64, error), remove func(context.Context, []int64) error, label string) (int, error) {
	deleted := 0
	for {
		ids, err := fetch(ctx)
		if err != nil {
			return deleted, fmt.Errorf("%s查询失败: %w", label, err)
		}
		if len(ids) == 0 {
			return deleted, nil
		}
		if err = remove(ctx, ids); err != nil {
			return deleted, fmt.Errorf("%s失败: %w", label, err)
		}
		deleted += len(ids)
	}
}
