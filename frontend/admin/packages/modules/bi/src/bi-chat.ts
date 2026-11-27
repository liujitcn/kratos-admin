import type { AiToolCall } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_tool";
import type { AskBiQueryResponse, BiQueryResult, BiQuerySpec, BiResultColumn, BiResultRow } from "./rpc/bi/admin/v1/bi_query";

const ASK_BI_QUERY_TOOL = "bi_admin_v1_bi_query_service_ask_bi_query";
const BI_CHAT_PREVIEW_ROWS = 6;

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

/** 旧版建卡块从同轮、同一查询编号的工具快照恢复结果预览。 */
export function resolveBiChatBlocks(blocks: BiChatBlock[], tools: AiToolCall[]): BiChatBlock[] {
  if (!blocks.some(block => block.type === "bi_card_guide")) return blocks;

  const results = new Map<number, BiQueryResult>();
  for (const tool of tools) {
    if (tool.name !== ASK_BI_QUERY_TOOL || tool.status !== "success") continue;
    try {
      const result = (JSON.parse(tool.output) as AskBiQueryResponse | null)?.result;
      if (result && Number.isSafeInteger(result.query_record_id)) results.set(result.query_record_id, result);
    } catch {
      // 旧的工具快照解析失败时，仍可展示已保存的建卡入口。
    }
  }

  return blocks.map(block => {
    if (block.type !== "bi_card_guide" || !block.query_record_id) return block;
    const result = results.get(block.query_record_id);
    if (!Array.isArray(result?.columns) || !result.columns.length || !Array.isArray(result.rows) || !result.rows.length) return block;

    const rows = result.rows.slice(0, BI_CHAT_PREVIEW_ROWS);
    if (!result.columns.every(column => typeof column?.code === "string") || !rows.every(row => Array.isArray(row?.values))) return block;

    return {
      ...block,
      type: "bi_query_result",
      columns: result.columns,
      rows,
      row_count: result.rows.length,
      truncated: result.truncated === true
    };
  });
}
