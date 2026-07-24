import service from "@/utils/request";
import {
  type CodeGenProtoService,
  type ListCodeGenProtoRequest,
  type ListCodeGenProtoResponse,
  type SaveCodeGenProtoRequest
} from "@/rpc/system/admin/v1/code_gen_proto";
import type { Empty } from "@/rpc/google/protobuf/empty";

const CODE_GEN_TABLE_URL = "/v1/admin/code-gen/table";

/** Admin代码生成Proto接口配置服务。 */
export class CodeGenProtoServiceImpl implements CodeGenProtoService {
  /** 查询代码生成Proto接口配置。 */
  ListCodeGenProto(request: ListCodeGenProtoRequest): Promise<ListCodeGenProtoResponse> {
    return service<ListCodeGenProtoRequest, ListCodeGenProtoResponse>({
      url: `${CODE_GEN_TABLE_URL}/${request.table_id}/proto`,
      method: "get"
    });
  }

  /** 保存代码生成Proto接口配置。 */
  SaveCodeGenProto(request: SaveCodeGenProtoRequest): Promise<Empty> {
    return service<SaveCodeGenProtoRequest, Empty>({
      url: `${CODE_GEN_TABLE_URL}/${request.table_id}/proto`,
      method: "put",
      data: request
    });
  }
}

export const defCodeGenProtoService = new CodeGenProtoServiceImpl();
