import service from "@liujitcn/kratos-admin-core/request";
import {
  type CodeGenColumnService,
  type ListCodeGenDatabaseColumnRequest,
  type ListCodeGenDatabaseColumnResponse,
  type ListCodeGenColumnRequest,
  type ListCodeGenColumnResponse,
  type SaveCodeGenColumnRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/code_gen_column";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import { normalizeCodeGenLocaleConfigMap, serializeCodeGenLocaleConfigMap } from "./code_gen_i18n";

const CODE_GEN_DATABASE_TABLE_URL = "/v1/admin/code-gen/database/table";
const CODE_GEN_TABLE_URL = "/v1/admin/code-gen/table";

/** Admin代码生成字段服务。 */
export class CodeGenColumnServiceImpl implements CodeGenColumnService {
  /** 查询数据库表字段列表。 */
  ListCodeGenDatabaseColumn(request: ListCodeGenDatabaseColumnRequest): Promise<ListCodeGenDatabaseColumnResponse> {
    return service<ListCodeGenDatabaseColumnRequest, ListCodeGenDatabaseColumnResponse>({
      url: `${CODE_GEN_DATABASE_TABLE_URL}/${request.table_name}/column`,
      method: "get"
    });
  }

  /** 查询代码生成字段配置。 */
  async ListCodeGenColumn(request: ListCodeGenColumnRequest): Promise<ListCodeGenColumnResponse> {
    const response = await service<ListCodeGenColumnRequest, ListCodeGenColumnResponse>({
      url: `${CODE_GEN_TABLE_URL}/${request.table_id}/column`,
      method: "get"
    });
    return {
      ...response,
      code_gen_columns: (response.code_gen_columns ?? []).map(column => ({
        ...column,
        i18n_config: normalizeCodeGenLocaleConfigMap(column.i18n_config)
      }))
    };
  }

  /** 保存代码生成字段配置。 */
  SaveCodeGenColumn(request: SaveCodeGenColumnRequest): Promise<Empty> {
    return service<SaveCodeGenColumnRequest, Empty>({
      url: `${CODE_GEN_TABLE_URL}/${request.table_id}/column`,
      method: "put",
      data: {
        ...request,
        code_gen_columns: request.code_gen_columns.map(column => ({
          ...column,
          i18n_config: serializeCodeGenLocaleConfigMap(column.i18n_config)
        }))
      }
    });
  }
}

export const defCodeGenColumnService = new CodeGenColumnServiceImpl();
