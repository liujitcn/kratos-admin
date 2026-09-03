import service from "@liujitcn/kratos-admin-core/request";
import type {
  CacheService,
  PageCacheRequest,
  PageCacheResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/cache";

const CACHE_URL = "/v1/admin/cache";

/** Admin运行时缓存查询服务。 */
export class CacheServiceImpl implements CacheService {
  /** 分页查询当前进程缓存条目。 */
  PageCache(request: PageCacheRequest): Promise<PageCacheResponse> {
    return service<PageCacheRequest, PageCacheResponse>({
      url: CACHE_URL,
      method: "get",
      params: request
    });
  }
}

export const defCacheService = new CacheServiceImpl();
