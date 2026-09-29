import service from "@liujitcn/kratos-admin-core/request";
import type {
  AiKnowledgeDocService,
  DeleteAiKnowledgeDocRequest,
  PageAiKnowledgeDocRequest,
  PageAiKnowledgeDocResponse,
  ReprocessAiKnowledgeDocRequest,
  UploadAiKnowledgeDocFileRequest,
  UploadAiKnowledgeDocTextRequest
} from "@liujitcn/kratos-admin-rag/rpc/rag/admin/v1/ai_knowledge_doc";
import type { Empty } from "@liujitcn/kratos-admin-rag/rpc/google/protobuf/empty";

const AI_KNOWLEDGE_URL = "/v1/admin/rag/knowledge";

/** AI知识库文档管理服务实现。 */
export class AiKnowledgeDocServiceImpl implements AiKnowledgeDocService {
  /** 分页查询知识库文档。 */
  PageAiKnowledgeDoc(request: PageAiKnowledgeDocRequest): Promise<PageAiKnowledgeDocResponse> {
    return service({ url: `${AI_KNOWLEDGE_URL}/${request.id}/doc`, method: "get", params: { page_num: request.page_num, page_size: request.page_size } });
  }

  /** 上传纯文本文档。 */
  UploadAiKnowledgeDocText(request: UploadAiKnowledgeDocTextRequest): Promise<Empty> {
    return service({ url: `${AI_KNOWLEDGE_URL}/${request.id}/doc/text`, method: "post", data: { name: request.name, content: request.content } });
  }

  /** 上传文档文件。 */
  UploadAiKnowledgeDocFile(request: UploadAiKnowledgeDocFileRequest): Promise<Empty> {
    return service({ url: `${AI_KNOWLEDGE_URL}/${request.id}/doc/file`, method: "post", data: request });
  }

  /** 删除知识库文档。 */
  DeleteAiKnowledgeDoc(request: DeleteAiKnowledgeDocRequest): Promise<Empty> {
    return service({ url: `${AI_KNOWLEDGE_URL}/doc/${request.doc_id}`, method: "delete" });
  }

  /** 重新处理失败的文档。 */
  ReprocessAiKnowledgeDoc(request: ReprocessAiKnowledgeDocRequest): Promise<Empty> {
    return service({ url: `${AI_KNOWLEDGE_URL}/doc/${request.doc_id}/reprocess`, method: "post" });
  }
}

export const defAiKnowledgeDocService = new AiKnowledgeDocServiceImpl();
