import service from "@/utils/request";
import {
  type CodeGenTableForm,
  type CodeGenTableService,
  type CreateCodeGenTableRequest,
  type DeleteCodeGenTableRequest,
  type GetCodeGenTableRequest,
  type ListCodeGenDatabaseTableRequest,
  type ListCodeGenDatabaseTableResponse,
  type PageCodeGenTableRequest,
  type PageCodeGenTableResponse,
  type UpdateCodeGenTableRequest
} from "@/rpc/system/admin/v1/code_gen_table";
import type { Empty } from "@/rpc/google/protobuf/empty";

const CODE_GEN_TABLE_URL = "/v1/admin/code-gen/table";
const CODE_GEN_DATABASE_TABLE_URL = "/v1/admin/code-gen/database/table";

/** Admin代码生成表配置服务。 */
export class CodeGenTableServiceImpl implements CodeGenTableService {
  /** 查询数据库表列表。 */
  ListCodeGenDatabaseTable(request: ListCodeGenDatabaseTableRequest): Promise<ListCodeGenDatabaseTableResponse> {
    return service<ListCodeGenDatabaseTableRequest, ListCodeGenDatabaseTableResponse>({
      url: CODE_GEN_DATABASE_TABLE_URL,
      method: "get",
      params: request
    });
  }

  /** 查询代码生成表配置分页列表。 */
  PageCodeGenTable(request: PageCodeGenTableRequest): Promise<PageCodeGenTableResponse> {
    return service<PageCodeGenTableRequest, PageCodeGenTableResponse>({
      url: CODE_GEN_TABLE_URL,
      method: "get",
      params: request
    });
  }

  /** 查询代码生成表配置。 */
  GetCodeGenTable(request: GetCodeGenTableRequest): Promise<CodeGenTableForm> {
    return service<GetCodeGenTableRequest, CodeGenTableForm>({
      url: `${CODE_GEN_TABLE_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建代码生成表配置。 */
  CreateCodeGenTable(request: CreateCodeGenTableRequest): Promise<Empty> {
    return service<CodeGenTableForm | undefined, Empty>({
      url: CODE_GEN_TABLE_URL,
      method: "post",
      data: request.code_gen_table
    });
  }

  /** 更新代码生成表配置。 */
  UpdateCodeGenTable(request: UpdateCodeGenTableRequest): Promise<Empty> {
    return service<CodeGenTableForm | undefined, Empty>({
      url: `${CODE_GEN_TABLE_URL}/${request.id}`,
      method: "put",
      data: request.code_gen_table
    });
  }

  /** 删除代码生成表配置。 */
  DeleteCodeGenTable(request: DeleteCodeGenTableRequest): Promise<Empty> {
    return service<DeleteCodeGenTableRequest, Empty>({
      url: `${CODE_GEN_TABLE_URL}/${request.ids}`,
      method: "delete"
    });
  }
}

export const defCodeGenTableService = new CodeGenTableServiceImpl();
