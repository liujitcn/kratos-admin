import service from "@liujitcn/kratos-admin/request";
import type {
  BaseMigrationService,
  BaseMigration,
  BaseMigrationListItem,
  GetBaseMigrationRequest,
  PageBaseMigrationRequest,
  PageBaseMigrationResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_migration";

const BASE_MIGRATION_URL = "/v1/admin/base-migration";

/** Admin数据库迁移服务。 */
export class BaseMigrationServiceImpl implements BaseMigrationService {
  /** 分页查询数据库升级历史。 */
  PageBaseMigration(request: PageBaseMigrationRequest): Promise<PageBaseMigrationResponse> {
    return service<PageBaseMigrationRequest, PageBaseMigrationResponse>({
      url: BASE_MIGRATION_URL,
      method: "get",
      params: request
    });
  }

  /** 查询数据库迁移记录详情。 */
  GetBaseMigration(request: GetBaseMigrationRequest): Promise<BaseMigration> {
    return service<GetBaseMigrationRequest, BaseMigration>({
      url: `/v1/admin/base-migration/${request.id}`,
      method: "get"
    });
  }
}

export const defBaseMigrationService = new BaseMigrationServiceImpl();

export type { BaseMigration, BaseMigrationListItem };
