import assert from "node:assert/strict";
import { test } from "node:test";
import { buildBiCsv, classifyBiCsvExportFailure } from "../src/bi-csv.ts";

test("导出保留真实列和值并阻止表格公式执行", () => {
  const csv = buildBiCsv({
    columns: [{ name: "@名称" }, { name: '单位"数量', type: "number" }, { code: "code" }],
    rows: [
      { values: [" =1+1", "-3.5", "a,b"] },
      { values: ["\t@SUM(A1)", "+1+2", 'a"b'] },
      { values: ["＝1", "9007199254740999", "line\nbreak"] }
    ]
  });
  assert.equal(csv, `"'@名称","单位""数量","code"\r\n"' =1+1","-3.5","a,b"\r\n"'\t@SUM(A1)","'+1+2","a""b"\r\n"'＝1","9007199254740999","line\nbreak"`);
});

test("只有临时的导出错误可重试，权限拒绝或不可用记录各自提示", () => {
  assert.equal(classifyBiCsvExportFailure({ response: { status: 403 } }), "denied");
  assert.equal(classifyBiCsvExportFailure({ response: { data: { reason: "PERMISSION_DENIED" } } }), "denied");
  assert.equal(classifyBiCsvExportFailure({ response: { status: 404, data: { reason: "RESOURCE_NOT_FOUND" } } }), "unavailable");
  assert.equal(classifyBiCsvExportFailure({ response: { status: 400 } }), "unavailable");
  assert.equal(classifyBiCsvExportFailure({ response: { status: 408 } }), "retry");
  assert.equal(classifyBiCsvExportFailure({ response: { status: 429 } }), "retry");
  assert.equal(classifyBiCsvExportFailure({ response: { status: 500 } }), "retry");
  assert.equal(classifyBiCsvExportFailure(new Error("Network Error")), "retry");
});
