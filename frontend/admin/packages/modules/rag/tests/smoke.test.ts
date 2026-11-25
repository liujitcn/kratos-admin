import assert from "node:assert";
import { readFile } from "node:fs/promises";
import { stripTypeScriptTypes } from "node:module";
import { join } from "node:path";
import { describe, it } from "node:test";
import { ref } from "vue";

describe("rag module smoke", () => {
  it("placeholder passes", () => {
    assert.ok(true);
  });

  it("text upload uses a Markdown editor and keeps the Markdown source", async () => {
    const source = await readFile(join(process.cwd(), "src/views/ai-knowledge/index.vue"), "utf8");

    assert.match(source, /import MarkdownEditor from "@liujitcn\/kratos-admin-core\/components\/MarkdownEditor\/index\.vue"/);
    assert.match(source, /prop: "content", label: [\s\S]*component: "slot", slotName: "content"/);
    assert.match(source, /<template #content>[\s\S]*<MarkdownEditor[\s\S]*v-model="textForm\.content"/);
    assert.doesNotMatch(source, /prop: "content", label: [\s\S]*component: "textarea"/);
  });

  it("uploads a selected file when Element Plus gives the list through a Vue proxy", async () => {
    const source = await readFile(join(process.cwd(), "src/views/ai-knowledge/index.vue"), "utf8");
    const body = source.match(/async function handleDocFileChange\([\s\S]*?\n\}/)?.[0];
    assert.ok(body);

    const uploads: unknown[] = [];
    const handleDocFileChange = new Function(
      "MAX_FILE_SIZE",
      "docsDialog",
      "docFileInputRef",
      "ElMessage",
      "t",
      "selectedExtractorConnectionId",
      "uploading",
      "uploadProgress",
      "readFileAsBase64",
      "defAiKnowledgeDocService",
      "loadDocs",
      "table",
      `${stripTypeScriptTypes(body)}; return handleDocFileChange;`
    )(
      10 * 1024 * 1024,
      { knowledge: { id: 1 } },
      { value: { clearFiles: () => undefined } },
      { warning: () => undefined, success: () => undefined, error: () => undefined },
      (key: string) => key,
      { value: 2 },
      { value: false },
      { current: 0, total: 0, name: "" },
      async () => "base64",
      { UploadAiKnowledgeDocFile: async (request: unknown) => uploads.push(request) },
      async () => undefined,
      { value: { getTableList: () => undefined } }
    );

    const files = [{ uid: 1, raw: new File(["test"], "test.pdf"), size: 4, name: "test.pdf" }];
    const uploadFiles = ref(files);
    await handleDocFileChange(files[0], uploadFiles.value);

    assert.equal(uploads.length, 1);
  });
});
