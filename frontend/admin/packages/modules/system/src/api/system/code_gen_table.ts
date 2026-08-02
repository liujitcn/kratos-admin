import service from "@liujitcn/kratos-admin-core/request";
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
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/code_gen_table";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import { normalizeCodeGenLocaleConfigMap, serializeCodeGenLocaleConfigMap } from "./code_gen_i18n";

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
  async GetCodeGenTable(request: GetCodeGenTableRequest): Promise<CodeGenTableForm> {
    const response = await service<GetCodeGenTableRequest, CodeGenTableForm>({
      url: `${CODE_GEN_TABLE_URL}/${request.id}`,
      method: "get"
    });
    return { ...response, i18n_config: normalizeCodeGenLocaleConfigMap(response.i18n_config) };
  }

  /** 创建代码生成表配置。 */
  CreateCodeGenTable(request: CreateCodeGenTableRequest): Promise<Empty> {
    return service<CodeGenTableForm | undefined, Empty>({
      url: CODE_GEN_TABLE_URL,
      method: "post",
      data: serializeCodeGenTableForm(request.code_gen_table)
    });
  }

  /** 更新代码生成表配置。 */
  UpdateCodeGenTable(request: UpdateCodeGenTableRequest): Promise<Empty> {
    return service<CodeGenTableForm | undefined, Empty>({
      url: `${CODE_GEN_TABLE_URL}/${request.id}`,
      method: "put",
      data: serializeCodeGenTableForm(request.code_gen_table)
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

/** 将表配置中的 Map 转为 REST JSON 对象。 */
function serializeCodeGenTableForm(value?: CodeGenTableForm) {
  if (!value) return value;
  return { ...value, i18n_config: serializeCodeGenLocaleConfigMap(value.i18n_config) };
}
