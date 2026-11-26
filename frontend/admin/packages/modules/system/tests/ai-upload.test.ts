import assert from "node:assert/strict";
import test from "node:test";
import { getAIImageMimeType, isAIGeneratedAttachment, planAIAttachmentUploads } from "../src/views/ai/chat/attachment.js";

test("AI 上传支持 PDF 和 XLSX，并按已有附件计算剩余数量", () => {
  const files = [new File(["pdf"], "notes.PDF"), new File(["sheet"], "metrics.XLSX"), new File(["data"], "records.csv")];

  const plan = planAIAttachmentUploads(files, [1, 1, 1, 1]);

  assert.deepEqual(plan.accepted, files.slice(0, 2));
  assert.deepEqual(plan.rejected, []);
  assert.equal(plan.remaining, 2);
  assert.equal(plan.truncated, true);
});

test("AI 上传保留单个文件的限制原因并按草稿累计大小", () => {
  const limit = 20 * 1024 * 1024;
  const oversized = new File([new Uint8Array(limit + 1)], "large.pdf");
  const unsupported = new File(["unsupported"], "notes.exe");
  const fits = new File(["1"], "notes.txt");
  const overTotal = new File(["2"], "metrics.csv");

  const plan = planAIAttachmentUploads([oversized, unsupported, fits, overTotal], [limit - 1]);

  assert.deepEqual(plan.accepted, [fits]);
  assert.deepEqual(plan.rejected, [
    { file: oversized, reason: "too_large" },
    { file: unsupported, reason: "unsupported" },
    { file: overTotal, reason: "total_too_large" }
  ]);
  assert.equal(plan.truncated, false);
});

test("AI 生成附件按持久目录识别，不把下载查询参数当成存储路径", () => {
  assert.equal(isAIGeneratedAttachment({ url: "/data/ai/generated/2026/report.csv" }), true);
  assert.equal(isAIGeneratedAttachment({ url: "ai/generated/2026/report.csv" }), true);
  assert.equal(isAIGeneratedAttachment({ url: "https://example.test/assets/data/ai/generated/report.csv?v=1" }), true);
  assert.equal(isAIGeneratedAttachment({ url: "/data/ai/2026/input.csv?name=/data/ai/generated/test.csv" }), false);
});

test("鉴权下载返回通用二进制类型时，图片预览仍使用正确的 MIME", () => {
  assert.equal(getAIImageMimeType({ name: "report.svg", mime_type: "application/octet-stream" }), "image/svg+xml");
  assert.equal(getAIImageMimeType({ name: "photo.JPG", mime_type: "" }), "image/jpeg");
  assert.equal(getAIImageMimeType({ name: "legacy", mime_type: "IMAGE/PNG; charset=utf-8" }), "image/png");
  assert.equal(getAIImageMimeType({ name: "report.pdf", mime_type: "application/octet-stream" }), undefined);
});
