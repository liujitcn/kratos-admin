import service from "@liujitcn/kratos-admin-core/request";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import type { BaseTableBackupRestoreService } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_backup_restore";
import type {
  BaseTableBackupRestore,
  ExecuteBaseTableBackupRestoreRequest,
  GetBaseTableBackupRestoreRequest,
  PageBaseTableBackupRestoreRequest,
  PageBaseTableBackupRestoreResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_backup_restore";

const BASE_TABLE_BACKUP_RESTORE_URL = "/v1/admin/base/table-backup-restore";

/** 数据库备份恢复服务实现。 */
export class BaseTableBackupRestoreServiceImpl implements BaseTableBackupRestoreService {
  /** 分页查询备份恢复记录。 */
  PageBaseTableBackupRestore(request: PageBaseTableBackupRestoreRequest): Promise<PageBaseTableBackupRestoreResponse> {
    return service<PageBaseTableBackupRestoreRequest, PageBaseTableBackupRestoreResponse>({
      url: BASE_TABLE_BACKUP_RESTORE_URL,
      method: "get",
      params: request
    });
  }

  /** 查询备份恢复记录。 */
  GetBaseTableBackupRestore(request: GetBaseTableBackupRestoreRequest): Promise<BaseTableBackupRestore> {
    return service<GetBaseTableBackupRestoreRequest, BaseTableBackupRestore>({
      url: `${BASE_TABLE_BACKUP_RESTORE_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 手工执行备份恢复。 */
  ExecuteBaseTableBackupRestore(request: ExecuteBaseTableBackupRestoreRequest): Promise<Empty> {
    return service<BaseTableBackupRestore | undefined, Empty>({
      url: BASE_TABLE_BACKUP_RESTORE_URL,
      method: "post",
      data: request.base_table_backup_restore
    });
  }
}

/** 数据库备份恢复服务实例。 */
export const defBaseTableBackupRestoreService = new BaseTableBackupRestoreServiceImpl();
