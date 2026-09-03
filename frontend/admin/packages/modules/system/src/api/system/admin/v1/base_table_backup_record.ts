import service from "@liujitcn/kratos-admin-core/request";
import type { BaseTableBackupRecordService } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_backup_record";
import type {
  BaseTableBackupRecord,
  GetBaseTableBackupRecordRequest,
  PageBaseTableBackupRecordRequest,
  PageBaseTableBackupRecordResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_backup_record";

const BASE_TABLE_BACKUP_RECORD_URL = "/v1/admin/base/table-backup-record";

/** 数据库备份记录服务实现。 */
export class BaseTableBackupRecordServiceImpl implements BaseTableBackupRecordService {
  /** 分页查询备份记录。 */
  PageBaseTableBackupRecord(request: PageBaseTableBackupRecordRequest): Promise<PageBaseTableBackupRecordResponse> {
    return service<PageBaseTableBackupRecordRequest, PageBaseTableBackupRecordResponse>({
      url: BASE_TABLE_BACKUP_RECORD_URL,
      method: "get",
      params: request
    });
  }

  /** 查询备份记录。 */
  GetBaseTableBackupRecord(request: GetBaseTableBackupRecordRequest): Promise<BaseTableBackupRecord> {
    return service<GetBaseTableBackupRecordRequest, BaseTableBackupRecord>({
      url: `${BASE_TABLE_BACKUP_RECORD_URL}/${request.id}`,
      method: "get"
    });
  }
}

/** 数据库备份记录服务实例。 */
export const defBaseTableBackupRecordService = new BaseTableBackupRecordServiceImpl();
