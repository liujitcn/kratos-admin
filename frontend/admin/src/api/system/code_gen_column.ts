import service from "@/utils/request";
import {
  type CodeGenColumnService,
  type ListCodeGenDatabaseColumnRequest,
  type ListCodeGenDatabaseColumnResponse,
  type ListCodeGenColumnRequest,
  type ListCodeGenColumnResponse,
  type SaveCodeGenColumnRequest
} from "@/rpc/system/admin/v1/code_gen_column";
import type { Empty } from "@/rpc/google/protobuf/empty";

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
  ListCodeGenColumn(request: ListCodeGenColumnRequest): Promise<ListCodeGenColumnResponse> {
    return service<ListCodeGenColumnRequest, ListCodeGenColumnResponse>({
      url: `${CODE_GEN_TABLE_URL}/${request.table_id}/column`,
      method: "get"
    });
  }

  /** 保存代码生成字段配置。 */
  SaveCodeGenColumn(request: SaveCodeGenColumnRequest): Promise<Empty> {
    return service<SaveCodeGenColumnRequest, Empty>({
      url: `${CODE_GEN_TABLE_URL}/${request.table_id}/column`,
      method: "put",
      data: request
    });
  }
}

export const defCodeGenColumnService = new CodeGenColumnServiceImpl();
