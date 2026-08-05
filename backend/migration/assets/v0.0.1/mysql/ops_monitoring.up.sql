UPDATE `base_menu`
SET `api` = '["/system.admin.v1.OpsMonitoringService/GetOpsRuntime", "/system.admin.v1.OpsMonitoringService/GetOpsTraffic", "/system.admin.v1.OpsMonitoringService/GetOpsServices", "/system.admin.v1.OpsMonitoringService/GetOpsStorage", "/system.admin.v1.OpsMonitoringService/GetOpsEndpoints", "/system.admin.v1.OpsMonitoringService/GetOpsNodes", "/system.admin.v1.OpsMonitoringService/GetOpsAlerts", "/base.v1.SseService/SubscribeSse"]'
WHERE `id` = 95008;
