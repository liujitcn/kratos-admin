import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseSession,
  ListCurrentBaseSessionsRequest,
  ListCurrentBaseSessionsResponse,
  GetCurrentBaseSessionRequest,
  PageOnlineBaseSessionsRequest,
  PageOnlineBaseSessionsResponse,
  RevokeAllBaseSessionsRequest,
  RevokeBaseSessionRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_session";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const SESSION_URL = "/v1/admin/base/session";

/** BaseSessionService Admin会话管理服务。 */
export class BaseSessionServiceImpl {
  /** 查询当前用户全部有效会话。 */
  ListCurrentBaseSessions(request: ListCurrentBaseSessionsRequest): Promise<ListCurrentBaseSessionsResponse> {
    return service<ListCurrentBaseSessionsRequest, ListCurrentBaseSessionsResponse>({
      url: `${SESSION_URL}/current-user`,
      method: "get",
      params: request
    });
  }

  /** 查询当前用户会话。 */
  GetCurrentBaseSession(request: GetCurrentBaseSessionRequest): Promise<BaseSession> {
    return service<GetCurrentBaseSessionRequest, BaseSession>({
      url: `${SESSION_URL}/current`,
      method: "get",
      params: request
    });
  }

  /** 撤销当前用户全部会话。 */
  RevokeAllBaseSessions(request: RevokeAllBaseSessionsRequest): Promise<Empty> {
    return service<RevokeAllBaseSessionsRequest, Empty>({
      url: `${SESSION_URL}/revoke-all`,
      method: "put",
      data: request
    });
  }

  /** 分页查询当前在线用户会话。 */
  PageOnlineBaseSessions(request: PageOnlineBaseSessionsRequest): Promise<PageOnlineBaseSessionsResponse> {
    return service<PageOnlineBaseSessionsRequest, PageOnlineBaseSessionsResponse>({
      url: `${SESSION_URL}/online`,
      method: "get",
      params: request
    });
  }

  /** 下线指定用户会话。 */
  RevokeBaseSession(request: RevokeBaseSessionRequest): Promise<Empty> {
    return service<RevokeBaseSessionRequest, Empty>({
      url: `${SESSION_URL}/${request.session_id}`,
      method: "delete"
    });
  }
}

/** defBaseSessionService Admin会话管理服务实例。 */
export const defBaseSessionService = new BaseSessionServiceImpl();
