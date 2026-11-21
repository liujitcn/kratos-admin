import service from "@liujitcn/kratos-admin-core/request";
import { normalizeBiQuerySpec, normalizeBiQueryResult } from "../../../../bi-response";
import type {
  AskBiQueryRequest,
  AskBiQueryResponse,
  BiQueryService,
  BiQueryResult,
  ListBiSourcesRequest,
  ListBiSourcesResponse,
  PreviewBiQueryRequest,
  RefineBiQueryRequest
} from "../../../../rpc/bi/admin/v1/bi_query";

const BI_QUERY_URL = "/v1/admin/bi";

/** BI 语义查询服务实现。 */
export class BiQueryServiceImpl implements BiQueryService {
  /** 查询当前用户可用的数据源目录。 */
  ListBiSources(request: ListBiSourcesRequest): Promise<ListBiSourcesResponse> {
    return service<ListBiSourcesRequest, ListBiSourcesResponse>({
      url: `${BI_QUERY_URL}/sources`,
      method: "get",
      params: request
    }).then(result => ({
      sources: (result.sources ?? []).map(source => ({
        ...source,
        dimensions: source.dimensions ?? [],
        measures: source.measures ?? []
      }))
    }));
  }

  /** 预览一个语义查询。 */
  PreviewBiQuery(request: PreviewBiQueryRequest): Promise<BiQueryResult> {
    return service<PreviewBiQueryRequest, BiQueryResult>({
      url: `${BI_QUERY_URL}/query/preview`,
      method: "post",
      data: request
    }).then(normalizeBiQueryResult);
  }

  /** 根据自然语言生成语义查询并预览。 */
  AskBiQuery(request: AskBiQueryRequest): Promise<AskBiQueryResponse> {
    return service<AskBiQueryRequest, AskBiQueryResponse>({
      url: `${BI_QUERY_URL}/query/ask`,
      method: "post",
      data: request
    }).then(normalizeAskResponse);
  }

  /** 调整已有问数的分组与时间范围。 */
  RefineBiQuery(request: RefineBiQueryRequest): Promise<AskBiQueryResponse> {
    return service<RefineBiQueryRequest, AskBiQueryResponse>({
      url: `${BI_QUERY_URL}/query/refine`,
      method: "post",
      data: request
    }).then(normalizeAskResponse);
  }
}

/** 补齐问数结果中省略的集合。 */
function normalizeAskResponse(response: AskBiQueryResponse): AskBiQueryResponse {
  return {
    ...response,
    query: response.query ? normalizeBiQuerySpec(response.query) : undefined,
    result: response.result ? normalizeBiQueryResult(response.result) : undefined
  };
}

export const defBiQueryService = new BiQueryServiceImpl();
