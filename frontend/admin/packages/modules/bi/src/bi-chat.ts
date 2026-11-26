import type { BiQuerySpec, BiResultColumn, BiResultRow } from "./rpc/bi/admin/v1/bi_query";

/** 智能问数建卡使用服务端记录和语义查询。 */
export interface BiCardGuide {
  query?: BiQuerySpec;
  query_record_id?: number;
  default_name?: string;
  chart_type?: string;
}

/** 指标澄清使用服务端核实后的数据源目录。 */
export interface BiMetricOption {
  code: string;
  name: string;
  unit?: string;
}

/** BI 模块处理的聊天结构化块。 */
export interface BiChatBlock extends BiCardGuide {
  type: "bi_query_result" | "bi_query_empty" | "bi_query_clarification" | "bi_permission_denied" | "bi_card_guide";
  columns?: BiResultColumn[];
  rows?: BiResultRow[];
  row_count?: number;
  truncated?: boolean;
  question?: string;
  source_code?: string;
  source_name?: string;
  options?: BiMetricOption[];
  code?: string;
}
