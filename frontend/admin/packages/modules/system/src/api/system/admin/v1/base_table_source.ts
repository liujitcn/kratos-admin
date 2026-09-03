import service from "@liujitcn/kratos-admin-core/request";
import type { StringValues } from "@liujitcn/kratos-admin-system/rpc/common/v1/types";
import type {
  BaseTableSourceService,
  OptionBaseTableRequest,
  OptionBaseTableSourceRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_source";

const BASE_TABLE_SOURCE_URL = "/v1/admin/base/table-source";

/** 数据源元数据服务实现。 */
export class BaseTableSourceServiceImpl implements BaseTableSourceService {
  /** 查询已初始化的数据源名称。 */
  OptionBaseTableSource(request: OptionBaseTableSourceRequest): Promise<StringValues> {
    return service<OptionBaseTableSourceRequest, StringValues>({
      url: `${BASE_TABLE_SOURCE_URL}/option`,
      method: "get",
      params: request
    });
  }

  /** 查询指定数据源中的数据库表名称。 */
  OptionBaseTable(request: OptionBaseTableRequest): Promise<StringValues> {
    return service<OptionBaseTableRequest, StringValues>({
      url: `${BASE_TABLE_SOURCE_URL}/table/option`,
      method: "get",
      params: request
    });
  }
}

/** 数据源元数据服务实例。 */
export const defBaseTableSourceService = new BaseTableSourceServiceImpl();
