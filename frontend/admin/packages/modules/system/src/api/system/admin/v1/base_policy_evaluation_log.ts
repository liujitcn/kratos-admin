import service from "@liujitcn/kratos-admin-core/request";
import type {
  BasePolicyEvaluationLog,
  BasePolicyEvaluationLogService,
  GetBasePolicyEvaluationLogRequest,
  PageBasePolicyEvaluationLogRequest,
  PageBasePolicyEvaluationLogResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_policy_evaluation_log";

const BASE_POLICY_EVALUATION_LOG_URL = "/v1/admin/base/policy-evaluation-log";

/** Admin 策略评估日志服务。 */
export class BasePolicyEvaluationLogServiceImpl implements BasePolicyEvaluationLogService {
  /** 查询策略评估日志分页列表。 */
  PageBasePolicyEvaluationLog(request: PageBasePolicyEvaluationLogRequest): Promise<PageBasePolicyEvaluationLogResponse> {
    return service<PageBasePolicyEvaluationLogRequest, PageBasePolicyEvaluationLogResponse>({
      url: BASE_POLICY_EVALUATION_LOG_URL,
      method: "get",
      params: request
    });
  }

  /** 查询策略评估日志详情。 */
  GetBasePolicyEvaluationLog(request: GetBasePolicyEvaluationLogRequest): Promise<BasePolicyEvaluationLog> {
    return service<GetBasePolicyEvaluationLogRequest, BasePolicyEvaluationLog>({
      url: `${BASE_POLICY_EVALUATION_LOG_URL}/${request.id}`,
      method: "get"
    });
  }
}

/** 默认策略评估日志服务。 */
export const defBasePolicyEvaluationLogService = new BasePolicyEvaluationLogServiceImpl();
