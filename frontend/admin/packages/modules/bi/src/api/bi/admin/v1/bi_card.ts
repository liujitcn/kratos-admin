import service from "@liujitcn/kratos-admin-core/request";
import { normalizeBiCard, normalizeBiWorkbench, normalizeBiQueryResult } from "../../../../bi-response";
import type {
  BiCard,
  BiCardService,
  BiWorkbench,
  DeleteBiCardRequest,
  DeleteBiCardResponse,
  GetBiCardDataRequest,
  GetBiWorkbenchRequest,
  PageBiCardRequest,
  PageBiCardResponse,
  SaveBiCardRequest,
  SaveBiWorkbenchRequest
} from "../../../../rpc/bi/admin/v1/bi_card";
import type { BiQueryResult } from "../../../../rpc/bi/admin/v1/bi_query";

const BI_CARD_URL = "/v1/admin/bi/card";
const BI_WORKBENCH_URL = "/v1/admin/bi/workbench";

/** 数据卡片与个人工作台请求实现。 */
export class BiCardServiceImpl implements BiCardService {
  /** 分页查询当前用户可见的卡片。 */
  PageBiCard(request: PageBiCardRequest): Promise<PageBiCardResponse> {
    return service<PageBiCardRequest, PageBiCardResponse>({ url: BI_CARD_URL, method: "get", params: request }).then(result => ({
      cards: (result.cards ?? []).map(normalizeBiCard),
      total: result.total ?? 0,
      counts: {
        all: result.counts?.all ?? 0,
        system: result.counts?.system ?? 0,
        mine: result.counts?.mine ?? 0,
        favorite: result.counts?.favorite ?? 0,
        shared: result.counts?.shared ?? 0
      }
    }));
  }
  /** 保存当前用户的数据卡片。 */
  SaveBiCard(request: SaveBiCardRequest): Promise<BiCard> {
    return service<SaveBiCardRequest, BiCard>({ url: BI_CARD_URL, method: "post", data: request }).then(normalizeBiCard);
  }
  /** 删除当前用户的数据卡片。 */
  DeleteBiCard(request: DeleteBiCardRequest): Promise<DeleteBiCardResponse> {
    return service({ url: `${BI_CARD_URL}/${request.id}`, method: "delete" });
  }
  /** 按访问者权限执行卡片查询。 */
  GetBiCardData(request: GetBiCardDataRequest): Promise<BiQueryResult> {
    return service<GetBiCardDataRequest, BiQueryResult>({
      url: `${BI_CARD_URL}/${request.id}/data`,
      method: "get",
      params: { range_days: request.range_days }
    }).then(normalizeBiQueryResult);
  }
  /** 获取个人工作台。 */
  GetBiWorkbench(request: GetBiWorkbenchRequest): Promise<BiWorkbench> {
    return service<GetBiWorkbenchRequest, BiWorkbench>({ url: BI_WORKBENCH_URL, method: "get", params: request }).then(
      normalizeBiWorkbench
    );
  }
  /** 保存个人工作台与收藏。 */
  SaveBiWorkbench(request: SaveBiWorkbenchRequest): Promise<BiWorkbench> {
    return service<SaveBiWorkbenchRequest, BiWorkbench>({ url: BI_WORKBENCH_URL, method: "put", data: request }).then(
      normalizeBiWorkbench
    );
  }
}

export const defBiCardService = new BiCardServiceImpl();
