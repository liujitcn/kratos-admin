import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { stripTypeScriptTypes } from "node:module";
import test from "node:test";
import { ref } from "vue";

test("Vue 代理文件列表按 UID 上传，单选与多选每批各上传一次", async () => {
  const source = await readFile("src/views/ai/chat/components/XSender.vue", "utf8");
  const body = source.match(/async function handleFileChange\([\s\S]*?\n\}/)?.[0];
  assert.ok(body);
  const batches: File[][] = [];
  const handleFileChange = new Function(
    "uploadAttachments",
    `${stripTypeScriptTypes(body)}; return handleFileChange;`
  )(async (files: File[]) => batches.push(files));
  for (const count of [1, 3]) {
    const files = Array.from({ length: count }, (_, index) => ({
      uid: index + 1,
      raw: new File(["test"], `test-${index}.pdf`)
    }));
    const uploadFiles = ref(files);
    assert.notEqual(files[0], uploadFiles.value[0]);
    for (const file of files) await handleFileChange(file, uploadFiles.value);
    assert.equal(batches.length, count === 1 ? 1 : 2);
    assert.deepEqual(batches.at(-1), files.map(file => file.raw));
  }
});
