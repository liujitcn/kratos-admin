import service from "@liujitcn/kratos-admin-core/request";
import type {
  GetProjectDocumentRequest,
  ProjectDocument,
  ProjectDocumentService,
  TreeProjectDocumentRequest,
  TreeProjectDocumentResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/project_document";

const PROJECT_DOCUMENT_URL = "/v1/admin/project-document";
const PROJECT_DOCUMENT_TREE_URL = `${PROJECT_DOCUMENT_URL}/tree`;

/** Admin项目文档服务。 */
export class ProjectDocumentServiceImpl implements ProjectDocumentService {
  /** 查询项目文档树。 */
  TreeProjectDocument(request: TreeProjectDocumentRequest): Promise<TreeProjectDocumentResponse> {
    return service<TreeProjectDocumentRequest, TreeProjectDocumentResponse>({
      url: PROJECT_DOCUMENT_TREE_URL,
      method: "get",
      params: request
    });
  }

  /** 查询项目文档详情。 */
  GetProjectDocument(request: GetProjectDocumentRequest): Promise<ProjectDocument> {
    return service<GetProjectDocumentRequest, ProjectDocument>({
      url: `${PROJECT_DOCUMENT_URL}/${request.id}`,
      method: "get"
    });
  }
}

export const defProjectDocumentService = new ProjectDocumentServiceImpl();
