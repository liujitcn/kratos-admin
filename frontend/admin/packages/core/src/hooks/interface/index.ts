export namespace Table {
  /** 表格分页状态。 */
  export interface Pageable {
    page_num: number;
    page_size: number;
    total: number;
  }
  /** useTable 维护的表格请求、分页和查询状态。 */
  export interface StateProps {
    loading: boolean;
    tableData: any[];
    pageable: Pageable;
    searchParam: {
      [key: string]: any;
    };
    searchInitParam: {
      [key: string]: any;
    };
    totalParam: {
      [key: string]: any;
    };
    icon?: {
      [key: string]: any;
    };
  }
}

export namespace HandleData {
  /** Element Plus 消息提示类型。 */
  export type MessageType = "" | "success" | "warning" | "info" | "error";
}

export namespace Theme {
  /** 后台菜单和头部主题类型。 */
  export type ThemeType = "light" | "inverted" | "dark";
  /** 灰色模式与色弱模式类型。 */
  export type GreyOrWeakType = "grey" | "weak";
}
