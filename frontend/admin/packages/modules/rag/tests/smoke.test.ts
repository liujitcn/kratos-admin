import assert from "node:assert";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { describe, it } from "node:test";

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
});
