import service from "@liujitcn/kratos-admin-core/request";
import type { BaseTableArchiveRecordService } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_archive_record";
import type {
  BaseTableArchiveRecord,
  GetBaseTableArchiveRecordRequest,
  PageBaseTableArchiveRecordRequest,
  PageBaseTableArchiveRecordResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_archive_record";

const BASE_TABLE_ARCHIVE_RECORD_URL = "/v1/admin/base/table-archive-record";

/** 表归档记录服务实现。 */
export class BaseTableArchiveRecordServiceImpl implements BaseTableArchiveRecordService {
  /** 分页查询归档记录。 */
  PageBaseTableArchiveRecord(request: PageBaseTableArchiveRecordRequest): Promise<PageBaseTableArchiveRecordResponse> {
    return service<PageBaseTableArchiveRecordRequest, PageBaseTableArchiveRecordResponse>({
      url: BASE_TABLE_ARCHIVE_RECORD_URL,
      method: "get",
      params: request
    });
  }

  /** 查询归档记录。 */
  GetBaseTableArchiveRecord(request: GetBaseTableArchiveRecordRequest): Promise<BaseTableArchiveRecord> {
    return service<GetBaseTableArchiveRecordRequest, BaseTableArchiveRecord>({
      url: `${BASE_TABLE_ARCHIVE_RECORD_URL}/${request.id}`,
      method: "get"
    });
  }
}

/** 表归档记录服务实例。 */
export const defBaseTableArchiveRecordService = new BaseTableArchiveRecordServiceImpl();
