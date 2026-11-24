import service from "@liujitcn/kratos-admin-core/request";
import type { StringValues } from "@liujitcn/kratos-admin-system/rpc/common/v1/types";
import type {
  DatabaseService,
  ListTableColumnsRequest,
  ListTableColumnsResponse,
  OptionBaseTableRequest,
  OptionBaseTableResponse,
  OptionBaseTableSourceRequest,
  TestConnectionRequest,
  TestConnectionResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/database";

const DATABASE_URL = "/v1/admin/database";

/** 管理端数据库服务。 */
export class DatabaseServiceImpl implements DatabaseService {
  /** 查询已初始化的数据源名称。 */
  OptionBaseTableSource(request: OptionBaseTableSourceRequest): Promise<StringValues> {
    return service({ url: `${DATABASE_URL}/sources`, method: "get", params: request });
  }

  /** 查询指定数据源中的数据库表选项。 */
  OptionBaseTable(request: OptionBaseTableRequest): Promise<OptionBaseTableResponse> {
    return service({ url: `${DATABASE_URL}/tables`, method: "get", params: request });
  }

  /** 查询指定数据表的字段。 */
  ListTableColumns(request: ListTableColumnsRequest): Promise<ListTableColumnsResponse> {
    return service({
      url: `${DATABASE_URL}/tables/${encodeURIComponent(request.table_name)}/columns`,
      method: "get",
      params: { source_name: request.source_name }
    });
  }

  /** 实际探测数据库连接是否可访问。 */
  TestConnection(request: TestConnectionRequest): Promise<TestConnectionResponse> {
    return service({ url: `${DATABASE_URL}/test-connection`, method: "post", data: request });
  }
}

/** 数据库服务实例。 */
export const defDatabaseService = new DatabaseServiceImpl();
