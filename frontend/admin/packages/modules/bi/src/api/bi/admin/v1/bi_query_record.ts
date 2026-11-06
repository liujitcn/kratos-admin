import service from "@liujitcn/kratos-admin-core/request";
import type {
  BiQueryRecordService,
  GetBiQueryRequest,
  PageBiQueryRequest,
  PageBiQueryResponse
} from "../../../../rpc/bi/admin/v1/bi_query_record";
import type { BiQueryRecord } from "../../../../rpc/bi/admin/v1/bi_query_record";

const BI_QUERY_URL = "/v1/admin/bi/query-record";

/** 智能问数记录管理服务实现。 */
export class BiQueryRecordServiceImpl implements BiQueryRecordService {
  /** 分页查询智能问数记录。 */
  PageBiQuery(request: PageBiQueryRequest): Promise<PageBiQueryResponse> {
    return service({ url: BI_QUERY_URL, method: "get", params: request });
  }

  /** 查询智能问数记录详情。 */
  GetBiQuery(request: GetBiQueryRequest): Promise<BiQueryRecord> {
    return service({ url: `${BI_QUERY_URL}/${request.id}`, method: "get" });
  }
}

export const defBiQueryRecordService = new BiQueryRecordServiceImpl();
