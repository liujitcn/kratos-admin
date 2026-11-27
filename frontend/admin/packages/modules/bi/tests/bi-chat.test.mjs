import assert from "node:assert/strict";
import { test } from "node:test";
import { resolveBiChatBlocks } from "../src/bi-chat.ts";

const askToolName = "bi_admin_v1_bi_query_service_ask_bi_query";
const guide = {
  type: "bi_card_guide",
  dialog: "bi_card_guide",
  query_record_id: 81,
  query: { source_code: "sales", dimensions: ["region"] },
  default_name: "区域销售额",
  chart_type: "bar"
};
const columns = [{ code: "region", name: "区域" }, { code: "amount", name: "销售额" }];
const rows = Array.from({ length: 8 }, (_, index) => ({ values: [`区域 ${index + 1}`, `900719925474099${index}`] }));

function savedAskResult(queryRecordId = 81) {
  return {
    name: askToolName,
    status: "success",
    output: JSON.stringify({ query: guide.query, result: { query_record_id: queryRecordId, columns, rows, truncated: true } })
  };
}

test("旧问数历史从同一查询编号的工具快照恢复六行预览", () => {
  const actual = resolveBiChatBlocks([guide], [savedAskResult(82), savedAskResult()]);
  assert.deepEqual(actual, [{
    ...guide,
    type: "bi_query_result",
    columns,
    rows: rows.slice(0, 6),
    row_count: 8,
    truncated: true
  }]);
  assert.equal(guide.type, "bi_card_guide");
});

test("缺失、非同轮查询结果或损坏的工具快照保留旧建卡入口", () => {
  const failed = { ...savedAskResult(), status: "error" };
  const otherTool = { ...savedAskResult(), name: "other_tool" };
  const brokenOutput = { ...savedAskResult(), output: '{"result":' };
  const invalidRow = { ...savedAskResult(), output: JSON.stringify({ result: { query_record_id: 81, columns, rows: [null] } }) };
  assert.deepEqual(resolveBiChatBlocks([guide], []), [guide]);
  assert.deepEqual(resolveBiChatBlocks([guide], [savedAskResult(82), failed, otherTool, brokenOutput, invalidRow]), [guide]);
});

test("原生结果卡优先显示已保存的结构化数据", () => {
  const native = [{ type: "bi_query_result", query_record_id: 81, columns, rows: rows.slice(2, 3) }];
  assert.equal(resolveBiChatBlocks(native, [savedAskResult()]), native);
});
