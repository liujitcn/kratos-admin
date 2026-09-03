import service from "@liujitcn/kratos-admin-core/request";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import type { BaseTableBackupService } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_backup";
import type {
  BaseTableBackupForm,
  CreateBaseTableBackupRequest,
  DeleteBaseTableBackupRequest,
  GetBaseTableBackupRequest,
  PageBaseTableBackupRequest,
  PageBaseTableBackupResponse,
  SetBaseTableBackupStatusRequest,
  UpdateBaseTableBackupRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_backup";

const BASE_TABLE_BACKUP_URL = "/v1/admin/base/table-backup";

/** 数据库备份配置服务实现。 */
export class BaseTableBackupServiceImpl implements BaseTableBackupService {
  /** 分页查询备份配置。 */
  PageBaseTableBackup(request: PageBaseTableBackupRequest): Promise<PageBaseTableBackupResponse> {
    return service<PageBaseTableBackupRequest, PageBaseTableBackupResponse>({
      url: BASE_TABLE_BACKUP_URL,
      method: "get",
      params: request
    });
  }

  /** 查询备份配置。 */
  GetBaseTableBackup(request: GetBaseTableBackupRequest): Promise<BaseTableBackupForm> {
    return service<GetBaseTableBackupRequest, BaseTableBackupForm>({
      url: `${BASE_TABLE_BACKUP_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建备份配置。 */
  CreateBaseTableBackup(request: CreateBaseTableBackupRequest): Promise<Empty> {
    return service<BaseTableBackupForm | undefined, Empty>({
      url: BASE_TABLE_BACKUP_URL,
      method: "post",
      data: request.base_table_backup
    });
  }

  /** 更新备份配置。 */
  UpdateBaseTableBackup(request: UpdateBaseTableBackupRequest): Promise<Empty> {
    return service<BaseTableBackupForm | undefined, Empty>({
      url: `${BASE_TABLE_BACKUP_URL}/${request.base_table_backup?.id ?? ""}`,
      method: "put",
      data: request.base_table_backup
    });
  }

  /** 删除备份配置。 */
  DeleteBaseTableBackup(request: DeleteBaseTableBackupRequest): Promise<Empty> {
    return service<DeleteBaseTableBackupRequest, Empty>({ url: `${BASE_TABLE_BACKUP_URL}/${request.id}`, method: "delete" });
  }

  /** 设置备份配置状态。 */
  SetBaseTableBackupStatus(request: SetBaseTableBackupStatusRequest): Promise<Empty> {
    return service<SetBaseTableBackupStatusRequest, Empty>({
      url: `${BASE_TABLE_BACKUP_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

/** 数据库备份配置服务实例。 */
export const defBaseTableBackupService = new BaseTableBackupServiceImpl();
