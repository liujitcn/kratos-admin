import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseApiLog,
  BaseDataAccessLog,
  BaseLoginLog,
  BaseOperationLog,
  BasePermissionLog,
  BasePolicyEvaluationLog,
  BaseApiLogService,
  BaseDataAccessLogService,
  BaseLoginLogService,
  BaseOperationLogService,
  BasePermissionLogService,
  BasePolicyEvaluationLogService,
  GetBaseApiLogRequest,
  GetBaseDataAccessLogRequest,
  GetBaseLoginLogRequest,
  GetBaseOperationLogRequest,
  GetBasePermissionLogRequest,
  GetBasePolicyEvaluationLogRequest,
  PageBaseApiLogRequest,
  PageBaseApiLogResponse,
  PageBaseDataAccessLogRequest,
  PageBaseDataAccessLogResponse,
  PageBaseLoginLogRequest,
  PageBaseLoginLogResponse,
  PageBaseOperationLogRequest,
  PageBaseOperationLogResponse,
  PageBasePermissionLogRequest,
  PageBasePermissionLogResponse,
  PageBasePolicyEvaluationLogRequest,
  PageBasePolicyEvaluationLogResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_audit_log";

const BASE_AUDIT_LOG_URL = "/v1/admin/base";

/** Admin 登录日志服务。 */
export class BaseLoginLogServiceImpl implements BaseLoginLogService {
  /** 查询登录日志分页列表。 */
  PageBaseLoginLog(request: PageBaseLoginLogRequest): Promise<PageBaseLoginLogResponse> {
    return service<PageBaseLoginLogRequest, PageBaseLoginLogResponse>({
      url: `${BASE_AUDIT_LOG_URL}/login-log`,
      method: "get",
      params: request
    });
  }

  /** 查询登录日志详情。 */
  GetBaseLoginLog(request: GetBaseLoginLogRequest): Promise<BaseLoginLog> {
    return service<GetBaseLoginLogRequest, BaseLoginLog>({
      url: `${BASE_AUDIT_LOG_URL}/login-log/${request.id}`,
      method: "get"
    });
  }
}

/** Admin API 访问日志服务。 */
export class BaseApiLogServiceImpl implements BaseApiLogService {
  /** 查询 API 访问日志分页列表。 */
  PageBaseApiLog(request: PageBaseApiLogRequest): Promise<PageBaseApiLogResponse> {
    return service<PageBaseApiLogRequest, PageBaseApiLogResponse>({
      url: `${BASE_AUDIT_LOG_URL}/api-log`,
      method: "get",
      params: request
    });
  }

  /** 查询 API 访问日志详情。 */
  GetBaseApiLog(request: GetBaseApiLogRequest): Promise<BaseApiLog> {
    return service<GetBaseApiLogRequest, BaseApiLog>({
      url: `${BASE_AUDIT_LOG_URL}/api-log/${request.id}`,
      method: "get"
    });
  }
}

/** Admin 业务操作日志服务。 */
export class BaseOperationLogServiceImpl implements BaseOperationLogService {
  /** 查询业务操作日志分页列表。 */
  PageBaseOperationLog(request: PageBaseOperationLogRequest): Promise<PageBaseOperationLogResponse> {
    return service<PageBaseOperationLogRequest, PageBaseOperationLogResponse>({
      url: `${BASE_AUDIT_LOG_URL}/operation-log`,
      method: "get",
      params: request
    });
  }

  /** 查询业务操作日志详情。 */
  GetBaseOperationLog(request: GetBaseOperationLogRequest): Promise<BaseOperationLog> {
    return service<GetBaseOperationLogRequest, BaseOperationLog>({
      url: `${BASE_AUDIT_LOG_URL}/operation-log/${request.id}`,
      method: "get"
    });
  }
}

/** Admin 数据访问日志服务。 */
export class BaseDataAccessLogServiceImpl implements BaseDataAccessLogService {
  /** 查询数据访问日志分页列表。 */
  PageBaseDataAccessLog(request: PageBaseDataAccessLogRequest): Promise<PageBaseDataAccessLogResponse> {
    return service<PageBaseDataAccessLogRequest, PageBaseDataAccessLogResponse>({
      url: `${BASE_AUDIT_LOG_URL}/data-access-log`,
      method: "get",
      params: request
    });
  }

  /** 查询数据访问日志详情。 */
  GetBaseDataAccessLog(request: GetBaseDataAccessLogRequest): Promise<BaseDataAccessLog> {
    return service<GetBaseDataAccessLogRequest, BaseDataAccessLog>({
      url: `${BASE_AUDIT_LOG_URL}/data-access-log/${request.id}`,
      method: "get"
    });
  }
}

/** Admin 权限日志服务。 */
export class BasePermissionLogServiceImpl implements BasePermissionLogService {
  /** 查询权限日志分页列表。 */
  PageBasePermissionLog(request: PageBasePermissionLogRequest): Promise<PageBasePermissionLogResponse> {
    return service<PageBasePermissionLogRequest, PageBasePermissionLogResponse>({
      url: `${BASE_AUDIT_LOG_URL}/permission-log`,
      method: "get",
      params: request
    });
  }

  /** 查询权限日志详情。 */
  GetBasePermissionLog(request: GetBasePermissionLogRequest): Promise<BasePermissionLog> {
    return service<GetBasePermissionLogRequest, BasePermissionLog>({
      url: `${BASE_AUDIT_LOG_URL}/permission-log/${request.id}`,
      method: "get"
    });
  }
}

/** Admin 策略评估日志服务。 */
export class BasePolicyEvaluationLogServiceImpl implements BasePolicyEvaluationLogService {
  /** 查询策略评估日志分页列表。 */
  PageBasePolicyEvaluationLog(request: PageBasePolicyEvaluationLogRequest): Promise<PageBasePolicyEvaluationLogResponse> {
    return service<PageBasePolicyEvaluationLogRequest, PageBasePolicyEvaluationLogResponse>({
      url: `${BASE_AUDIT_LOG_URL}/policy-evaluation-log`,
      method: "get",
      params: request
    });
  }

  /** 查询策略评估日志详情。 */
  GetBasePolicyEvaluationLog(request: GetBasePolicyEvaluationLogRequest): Promise<BasePolicyEvaluationLog> {
    return service<GetBasePolicyEvaluationLogRequest, BasePolicyEvaluationLog>({
      url: `${BASE_AUDIT_LOG_URL}/policy-evaluation-log/${request.id}`,
      method: "get"
    });
  }
}

/** 默认登录日志服务。 */
export const defBaseLoginLogService = new BaseLoginLogServiceImpl();
/** 默认 API 访问日志服务。 */
export const defBaseApiLogService = new BaseApiLogServiceImpl();
/** 默认业务操作日志服务。 */
export const defBaseOperationLogService = new BaseOperationLogServiceImpl();
/** 默认数据访问日志服务。 */
export const defBaseDataAccessLogService = new BaseDataAccessLogServiceImpl();
/** 默认权限日志服务。 */
export const defBasePermissionLogService = new BasePermissionLogServiceImpl();
/** 默认策略评估日志服务。 */
export const defBasePolicyEvaluationLogService = new BasePolicyEvaluationLogServiceImpl();
