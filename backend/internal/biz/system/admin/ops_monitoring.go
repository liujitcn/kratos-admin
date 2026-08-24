package biz

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/resource/i18n"

	"github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/utils"
	"github.com/redis/go-redis/v9"
)

const (
	defaultMonitoringWindow = 15
	monitoringBucketCount   = 12
	monitoringLogLimit      = 10000
)

// OpsMonitoringCase 提供当前服务运行状态和访问日志聚合能力。
type OpsMonitoringCase struct {
	*biz.BaseCase
	baseLogRepository *data.BaseLogRepository
	catalog           *i18n.I18n
}

// NewOpsMonitoringCase 创建运维监控业务实例。
func NewOpsMonitoringCase(
	baseCase *biz.BaseCase,
	baseLogRepository *data.BaseLogRepository,
	catalog *i18n.I18n,
) *OpsMonitoringCase {
	return &OpsMonitoringCase{
		BaseCase:          baseCase,
		baseLogRepository: baseLogRepository,
		catalog:           catalog,
	}
}

// GetOpsRuntime 查询当前进程运行信息。
func (c *OpsMonitoringCase) GetOpsRuntime(ctx context.Context, _ *adminv1.GetOpsRuntimeRequest) *adminv1.OpsRuntime {
	return c.runtimeInfo(time.Now(), biz.LocaleFromContext(ctx))
}

// GetOpsTraffic 查询请求流量与延迟趋势。
func (c *OpsMonitoringCase) GetOpsTraffic(ctx context.Context, req *adminv1.GetOpsTrafficRequest) (*adminv1.OpsTrafficResponse, error) {
	windowMinutes := monitoringWindow(req.GetWindowMinutes())
	now := time.Now()
	start := now.Add(-time.Duration(windowMinutes) * time.Minute)
	logs, err := c.loadLogs(ctx, start)
	if err != nil {
		return nil, err
	}
	return &adminv1.OpsTrafficResponse{
		CollectedAt:   now.Format(time.RFC3339Nano),
		WindowMinutes: int32(windowMinutes),
		Traffic:       summarizeTraffic(logs, windowMinutes),
		Points:        summarizeTrafficPoints(logs, start, now),
	}, nil
}

// GetOpsServices 查询服务和外部依赖状态。
func (c *OpsMonitoringCase) GetOpsServices(ctx context.Context, _ *adminv1.GetOpsServicesRequest) *adminv1.OpsServicesResponse {
	services, _ := c.collectDependencies(ctx)
	return &adminv1.OpsServicesResponse{CollectedAt: time.Now().Format(time.RFC3339Nano), Services: services}
}

// GetOpsStorage 查询数据库和缓存状态。
func (c *OpsMonitoringCase) GetOpsStorage(ctx context.Context, _ *adminv1.GetOpsStorageRequest) *adminv1.OpsStorageResponse {
	_, storage := c.collectDependencies(ctx)
	return &adminv1.OpsStorageResponse{CollectedAt: time.Now().Format(time.RFC3339Nano), Storage: storage}
}

// GetOpsEndpoints 查询接口请求摘要。
func (c *OpsMonitoringCase) GetOpsEndpoints(ctx context.Context, req *adminv1.GetOpsEndpointsRequest) (*adminv1.OpsEndpointsResponse, error) {
	windowMinutes := monitoringWindow(req.GetWindowMinutes())
	now := time.Now()
	logs, err := c.loadLogs(ctx, now.Add(-time.Duration(windowMinutes)*time.Minute))
	if err != nil {
		return nil, err
	}
	return &adminv1.OpsEndpointsResponse{
		CollectedAt:   now.Format(time.RFC3339Nano),
		WindowMinutes: int32(windowMinutes),
		Endpoints:     summarizeEndpoints(logs, windowMinutes, c.catalog, biz.LocaleFromContext(ctx)),
	}, nil
}

// GetOpsNodes 查询实例资源状态。
func (c *OpsMonitoringCase) GetOpsNodes(ctx context.Context, _ *adminv1.GetOpsNodesRequest) *adminv1.OpsNodesResponse {
	now := time.Now()
	locale := biz.LocaleFromContext(ctx)
	return &adminv1.OpsNodesResponse{CollectedAt: now.Format(time.RFC3339Nano), Nodes: summarizeNodes(c.runtimeInfo(now, locale), collectNodeMetrics(c.catalog, locale), c.catalog, locale)}
}

// GetOpsAlerts 查询窗口内告警事件。
func (c *OpsMonitoringCase) GetOpsAlerts(ctx context.Context, req *adminv1.GetOpsAlertsRequest) (*adminv1.OpsAlertsResponse, error) {
	windowMinutes := monitoringWindow(req.GetWindowMinutes())
	now := time.Now()
	logs, err := c.loadLogs(ctx, now.Add(-time.Duration(windowMinutes)*time.Minute))
	if err != nil {
		return nil, err
	}
	return &adminv1.OpsAlertsResponse{
		CollectedAt:   now.Format(time.RFC3339Nano),
		WindowMinutes: int32(windowMinutes),
		Alerts:        summarizeAlerts(logs, c.catalog, biz.LocaleFromContext(ctx)),
	}, nil
}

// loadLogs 查询统计窗口内的访问日志，并限制单次聚合规模。
func (c *OpsMonitoringCase) loadLogs(ctx context.Context, start time.Time) ([]dto.OpsLogRecord, error) {
	query := c.baseLogRepository.Query(ctx).BaseLog
	items, err := query.WithContext(ctx).Where(query.RequestTime.Gte(start)).Order(query.RequestTime.Desc()).Limit(monitoringLogLimit).Find()
	if err != nil {
		return nil, err
	}
	logs := make([]dto.OpsLogRecord, 0, len(items))
	for _, item := range items {
		logs = append(logs, toOpsLogRecord(item))
	}
	return logs, nil
}

// runtimeInfo 读取当前 Go 进程的运行时信息。
func (c *OpsMonitoringCase) runtimeInfo(now time.Time, locale string) *adminv1.OpsRuntime {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	appInfo := c.GetAppInfo()
	startTime := appInfo.GetStartTime()
	uptimeSeconds := int64(0)
	startAt := now
	if startTime != nil {
		startAt = startTime.AsTime()
		uptimeSeconds = int64(now.Sub(startAt).Seconds())
		if uptimeSeconds < 0 {
			uptimeSeconds = 0
		}
	}
	return &adminv1.OpsRuntime{
		ServiceName:      firstNonEmpty(appInfo.GetName(), appInfo.GetAppId(), monitoringText(c.catalog, locale, "unnamed_service")),
		Version:          appInfo.GetVersion(),
		Hostname:         firstNonEmpty(appInfo.GetHostname(), hostname(), monitoringText(c.catalog, locale, "current_instance")),
		Environment:      appInfo.GetEnvironment(),
		GoVersion:        runtime.Version(),
		Os:               runtime.GOOS,
		Arch:             runtime.GOARCH,
		StartTime:        startAt.Format(time.RFC3339Nano),
		UptimeSeconds:    uptimeSeconds,
		Goroutines:       uint64(runtime.NumGoroutine()),
		MemoryAllocBytes: memory.Alloc,
		MemorySysBytes:   memory.Sys,
	}
}

// collectDependencies 采集数据库和 Redis 的连接状态与连接池信息。
func (c *OpsMonitoringCase) collectDependencies(ctx context.Context) ([]*adminv1.OpsServiceStatus, []*adminv1.OpsStorage) {
	locale := biz.LocaleFromContext(ctx)
	appInfo := c.GetAppInfo()
	services := []*adminv1.OpsServiceStatus{
		{Name: "Backend API", Address: appInfo.GetAppId(), Status: monitoringText(c.catalog, locale, "normal"), Message: monitoringText(c.catalog, locale, "monitor_healthy")},
	}
	storage := make([]*adminv1.OpsStorage, 0, 2)
	databaseService, databaseStorage := c.databaseStatus(ctx)
	services = append(services, databaseService)
	if databaseStorage != nil {
		storage = append(storage, databaseStorage)
	}
	redisService, redisStorage := c.redisStatus(ctx)
	services = append(services, redisService)
	if redisStorage != nil {
		storage = append(storage, redisStorage)
	}
	return services, storage
}

// databaseStatus 检查数据库连接并读取连接池统计。
func (c *OpsMonitoringCase) databaseStatus(ctx context.Context) (*adminv1.OpsServiceStatus, *adminv1.OpsStorage) {
	locale := biz.LocaleFromContext(ctx)
	dataConfig := c.GetConfig().GetData()
	config := dataConfig.GetDatabase()
	if config == nil {
		config = dataConfig.GetDatabases()[gorm.DefaultClientName]
	}
	database := c.GormClients[gorm.DefaultClientName]
	address := databaseAddress(config.GetDriver(), config.GetSource(), c.catalog, locale)
	service := &adminv1.OpsServiceStatus{Name: "MySQL", Address: address, Status: monitoringText(c.catalog, locale, "error")}
	storage := &adminv1.OpsStorage{
		Name:          "MySQL · primary",
		ShortName:     "SQL",
		Address:       address,
		Status:        monitoringText(c.catalog, locale, "error"),
		CapacityLabel: monitoringText(c.catalog, locale, "pool"),
	}
	sqlDB, err := database.DB.DB()
	if err != nil {
		service.Message = err.Error()
		storage.Metrics = []*adminv1.OpsMetric{{Label: monitoringText(c.catalog, locale, "pool"), Value: monitoringText(c.catalog, locale, "unavailable")}}
		return service, storage
	}
	startedAt := time.Now()
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	pingErr := sqlDB.PingContext(pingCtx)
	cancel()
	service.LatencyMs = time.Since(startedAt).Milliseconds()
	stats := sqlDB.Stats()
	maxOpen := stats.MaxOpenConnections
	capacity := 0.0
	if maxOpen > 0 {
		capacity = float64(stats.InUse) / float64(maxOpen) * 100
	}
	storage.Capacity = capacity
	storage.Metrics = []*adminv1.OpsMetric{
		{Label: monitoringText(c.catalog, locale, "active_connections"), Value: fmt.Sprintf("%d / %d", stats.InUse, maxOpen)},
		{Label: monitoringText(c.catalog, locale, "idle_connections"), Value: fmt.Sprintf("%d", stats.Idle)},
		{Label: monitoringText(c.catalog, locale, "wait_count"), Value: fmt.Sprintf("%d", stats.WaitCount)},
	}
	if pingErr == nil {
		service.Status = monitoringText(c.catalog, locale, "normal")
		service.Message = monitoringText(c.catalog, locale, "connection_healthy")
		storage.Status = monitoringText(c.catalog, locale, "normal")
	} else {
		service.Message = pingErr.Error()
	}
	return service, storage
}

// redisStatus 检查 Redis 配置和连接状态。
func (c *OpsMonitoringCase) redisStatus(ctx context.Context) (*adminv1.OpsServiceStatus, *adminv1.OpsStorage) {
	locale := biz.LocaleFromContext(ctx)
	dataConfig := c.GetConfig().GetData()
	if dataConfig == nil {
		return &adminv1.OpsServiceStatus{Name: "Redis", Status: monitoringText(c.catalog, locale, "unconfigured"), Message: monitoringText(c.catalog, locale, "redis_unconfigured")}, nil
	}
	config := dataConfig.GetRedis()
	if config == nil || len(config.GetAddr()) == 0 {
		return &adminv1.OpsServiceStatus{Name: "Redis", Status: monitoringText(c.catalog, locale, "unconfigured"), Message: monitoringText(c.catalog, locale, "redis_unconfigured")}, nil
	}
	address := strings.Join(config.GetAddr(), ", ")
	service := &adminv1.OpsServiceStatus{Name: "Redis", Address: address, Status: monitoringText(c.catalog, locale, "error")}
	storage := &adminv1.OpsStorage{
		Name:          "Redis · cache",
		ShortName:     "RED",
		Address:       address,
		Status:        monitoringText(c.catalog, locale, "error"),
		CapacityLabel: monitoringText(c.catalog, locale, "connection"),
		Capacity:      0,
		Metrics:       []*adminv1.OpsMetric{{Label: monitoringText(c.catalog, locale, "database"), Value: fmt.Sprintf("%d", config.GetDb())}},
	}
	options, err := utils.GetUniversalOptions(config)
	if err != nil {
		service.Message = err.Error()
		return service, storage
	}
	client := redis.NewUniversalClient(options)
	startedAt := time.Now()
	pingErr := client.Ping(ctx).Err()
	closeErr := client.Close()
	service.LatencyMs = time.Since(startedAt).Milliseconds()
	if pingErr == nil && closeErr == nil {
		service.Status = monitoringText(c.catalog, locale, "normal")
		service.Message = monitoringText(c.catalog, locale, "connection_healthy")
		storage.Status = monitoringText(c.catalog, locale, "normal")
		storage.Metrics = append(storage.Metrics, &adminv1.OpsMetric{Label: monitoringText(c.catalog, locale, "address_count"), Value: fmt.Sprintf("%d", len(config.GetAddr()))})
		return service, storage
	}
	if pingErr != nil {
		service.Message = pingErr.Error()
	} else {
		service.Message = closeErr.Error()
	}
	return service, storage
}

// summarizeTraffic 聚合窗口内的请求摘要。
func summarizeTraffic(logs []dto.OpsLogRecord, windowMinutes int) *adminv1.OpsTraffic {
	costs := make([]int64, 0, len(logs))
	errors := 0
	for _, log := range logs {
		costs = append(costs, log.CostTime)
		if !log.IsSuccess {
			errors++
		}
	}
	requestCount := len(logs)
	errorRate := 0.0
	if requestCount > 0 {
		errorRate = float64(errors) / float64(requestCount) * 100
	}
	return &adminv1.OpsTraffic{
		Qps:          float64(requestCount) / float64(windowMinutes*60),
		P95LatencyMs: percentile(costs, 0.95),
		ErrorRate:    errorRate,
		Availability: 100 - errorRate,
		RequestCount: int64(requestCount),
	}
}

// summarizeTrafficPoints 按固定数量的时间桶聚合请求趋势。
func summarizeTrafficPoints(logs []dto.OpsLogRecord, start time.Time, end time.Time) []*adminv1.OpsTrafficPoint {
	bucketDuration := end.Sub(start) / monitoringBucketCount
	buckets := make([]dto.OpsMinuteAggregate, monitoringBucketCount)
	for index := range buckets {
		buckets[index].At = start.Add(time.Duration(index) * bucketDuration)
	}
	for _, log := range logs {
		index := int(log.RequestTime.Sub(start) / bucketDuration)
		if index < 0 || index >= len(buckets) {
			continue
		}
		buckets[index].Total++
		buckets[index].Costs = append(buckets[index].Costs, log.CostTime)
	}
	maxQPS := 0.0
	maxLatency := 0.0
	points := make([]*adminv1.OpsTrafficPoint, 0, len(buckets))
	for _, bucket := range buckets {
		qps := float64(bucket.Total) / bucketDuration.Seconds()
		latency := percentile(bucket.Costs, 0.95)
		if qps > maxQPS {
			maxQPS = qps
		}
		if latency > maxLatency {
			maxLatency = latency
		}
		points = append(points, &adminv1.OpsTrafficPoint{At: bucket.At.Format(time.RFC3339Nano), Qps: qps, P95LatencyMs: latency})
	}
	for _, point := range points {
		if maxQPS > 0 {
			point.QpsPercent = point.Qps / maxQPS * 100
		}
		if maxLatency > 0 {
			point.LatencyPercent = point.P95LatencyMs / maxLatency * 100
		}
	}
	return points
}

// summarizeEndpoints 按接口路径聚合窗口内的请求指标。
func summarizeEndpoints(logs []dto.OpsLogRecord, windowMinutes int, catalog *i18n.I18n, locale string) []*adminv1.OpsEndpoint {
	aggregates := make(map[string]*dto.OpsEndpointAggregate)
	for _, log := range logs {
		if log.Path == "" {
			continue
		}
		aggregate, exists := aggregates[log.Path]
		if !exists {
			aggregate = &dto.OpsEndpointAggregate{Route: log.Path}
			aggregates[log.Path] = aggregate
		}
		aggregate.Total++
		aggregate.Costs = append(aggregate.Costs, log.CostTime)
		if !log.IsSuccess {
			aggregate.Errors++
		}
	}
	items := make([]*dto.OpsEndpointAggregate, 0, len(aggregates))
	for _, aggregate := range aggregates {
		items = append(items, aggregate)
	}
	sort.Slice(items, func(left int, right int) bool { return items[left].Total > items[right].Total })
	if len(items) > 8 {
		items = items[:8]
	}
	endpoints := make([]*adminv1.OpsEndpoint, 0, len(items))
	for _, item := range items {
		errorRate := float64(item.Errors) / float64(item.Total) * 100
		latency := percentile(item.Costs, 0.95)
		status := monitoringText(catalog, locale, "normal")
		if item.Errors > 0 || latency >= 500 {
			status = monitoringText(catalog, locale, "attention")
		}
		endpoints = append(endpoints, &adminv1.OpsEndpoint{
			Route:        item.Route,
			Qps:          float64(item.Total) / float64(windowMinutes*60),
			P95LatencyMs: latency,
			ErrorRate:    errorRate,
			Status:       status,
		})
	}
	return endpoints
}

// summarizeNodes 将进程和系统资源使用情况映射为实例资源指标。
func summarizeNodes(runtimeInfo *adminv1.OpsRuntime, metrics []*adminv1.OpsNodeMetric, catalog *i18n.I18n, locale string) []*adminv1.OpsNode {
	usedPercent := 0.0
	if runtimeInfo.GetMemorySysBytes() > 0 {
		usedPercent = float64(runtimeInfo.GetMemoryAllocBytes()) / float64(runtimeInfo.GetMemorySysBytes()) * 100
	}
	metrics = append([]*adminv1.OpsNodeMetric{{
		Label:      monitoringText(catalog, locale, "heap_memory"),
		Value:      math.Min(usedPercent, 100),
		UsedBytes:  runtimeInfo.GetMemoryAllocBytes(),
		TotalBytes: runtimeInfo.GetMemorySysBytes(),
	}}, metrics...)
	return []*adminv1.OpsNode{{
		Name:    runtimeInfo.GetHostname(),
		Metrics: metrics,
	}}
}

// collectNodeMetrics 尽力采集实例可见的物理内存和程序所在文件系统使用率，单项失败时保留其他可用指标。
func collectNodeMetrics(catalog *i18n.I18n, locale string) []*adminv1.OpsNodeMetric {
	metrics := make([]*adminv1.OpsNodeMetric, 0, 2)
	memoryStat, err := mem.VirtualMemory()
	if err == nil && memoryStat.Total > 0 {
		metrics = append(metrics, &adminv1.OpsNodeMetric{
			Label:      monitoringText(catalog, locale, "memory"),
			Value:      math.Min(memoryStat.UsedPercent, 100),
			UsedBytes:  memoryStat.Used,
			TotalBytes: memoryStat.Total,
		})
	}
	diskPath := "."
	var executablePath string
	executablePath, err = os.Executable()
	if err == nil {
		diskPath = filepath.Dir(executablePath)
	}
	var diskStat *disk.UsageStat
	diskStat, err = disk.Usage(diskPath)
	if err == nil && diskStat.Total > 0 {
		metrics = append(metrics, &adminv1.OpsNodeMetric{
			Label:      monitoringText(catalog, locale, "disk"),
			Value:      math.Min(diskStat.UsedPercent, 100),
			UsedBytes:  diskStat.Used,
			TotalBytes: diskStat.Total,
		})
	}
	return metrics
}

// summarizeAlerts 将窗口内失败请求转换为告警事件。
func summarizeAlerts(logs []dto.OpsLogRecord, catalog *i18n.I18n, locale string) []*adminv1.OpsAlert {
	alerts := make([]*adminv1.OpsAlert, 0)
	for _, log := range logs {
		if log.IsSuccess {
			continue
		}
		statusCode := log.StatusCode
		if statusCode == 0 {
			statusCode = 500
		}
		alerts = append(alerts, &adminv1.OpsAlert{
			Title:  monitoringText(catalog, locale, "request_failed"),
			Detail: fmt.Sprintf("%s · HTTP %d · %s %d ms", firstNonEmpty(log.Path, monitoringText(catalog, locale, "unknown_endpoint")), statusCode, monitoringText(catalog, locale, "duration"), log.CostTime),
			At:     log.RequestTime.Format(time.RFC3339Nano),
			Status: monitoringText(catalog, locale, "unresolved"),
		})
		if len(alerts) >= 8 {
			break
		}
	}
	return alerts
}

// toOpsLogRecord 将访问日志模型转换为监控聚合记录。
func toOpsLogRecord(item *models.BaseLog) dto.OpsLogRecord {
	return dto.OpsLogRecord{Path: item.Path, RequestTime: item.RequestTime, CostTime: item.CostTime, IsSuccess: item.IsSuccess, StatusCode: item.StatusCode}
}

// percentile 计算有序样本的近似百分位数。
func percentile(values []int64, ratio float64) float64 {
	if len(values) == 0 {
		return 0
	}
	values = append([]int64(nil), values...)
	sort.Slice(values, func(left int, right int) bool { return values[left] < values[right] })
	index := int(math.Ceil(float64(len(values))*ratio)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return float64(values[index])
}

// databaseAddress 返回不包含账号密码的数据库地址摘要。
func databaseAddress(driver string, source string, catalog *i18n.I18n, locale string) string {
	if index := strings.Index(source, "@tcp("); index >= 0 {
		end := strings.Index(source[index+len("@tcp("):], ")")
		if end >= 0 {
			return source[index+len("@tcp(") : index+len("@tcp(")+end]
		}
	}
	if driver != "" {
		return driver
	}
	return monitoringText(catalog, locale, "undeclared")
}

// hostname 返回当前主机名，读取失败时返回空字符串。
func hostname() string {
	name, err := os.Hostname()
	if err != nil || name == "" {
		return ""
	}
	return name
}

// firstNonEmpty 返回第一个非空字符串，不存在时返回空字符串。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// monitoringText 返回运维监控字段的当前语言文案。
func monitoringText(catalog *i18n.I18n, locale, key string) string {
	messageKey := "system.admin.ops_monitoring." + key
	return catalog.Localize(locale, "zh-CN", messageKey, nil, messageKey)
}

// monitoringWindow 返回有效的监控统计窗口。
func monitoringWindow(windowMinutes int32) int {
	if windowMinutes < 1 || windowMinutes > 60 {
		return defaultMonitoringWindow
	}
	return int(windowMinutes)
}
