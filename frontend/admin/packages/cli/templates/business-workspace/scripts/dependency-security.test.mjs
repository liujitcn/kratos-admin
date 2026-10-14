import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";

const require = createRequire(import.meta.url);

for (const consumer of ["stylelint", "lint-staged"]) {
  const consumerRequire = createRequire(require.resolve(consumer));
  const micromatchRequire = createRequire(consumerRequire.resolve("micromatch"));
  const braces = micromatchRequire("braces");

  test(`${consumer} 保留常规嵌套 glob 的编译与展开结果`, () => {
    const pattern = "src/{core,modules}/{a,b}.vue";
    assert.equal(braces.compile(pattern), "src/(core|modules)/(a|b).vue");
    assert.deepEqual(braces.expand(pattern), ["src/core/a.vue", "src/core/b.vue", "src/modules/a.vue", "src/modules/b.vue"]);
    assert.equal(braces.stringify(pattern), pattern);
    assert.deepEqual(braces.expand("{1..3}"), ["1", "2", "3"]);
  });

  for (const [name, pattern] of [
    ["花括号", "{".repeat(4000) + "a,b" + "}".repeat(4000)],
    ["圆括号", "(".repeat(4000) + "a" + ")".repeat(4000)],
    ["未闭合括号", "{".repeat(4000) + "a"]
  ]) {
    test(`${consumer} 在递归前拒绝过深${name}`, () => {
      for (const method of ["parse", "compile", "expand", "stringify"]) {
        assert.throws(() => braces[method](pattern), { name: "SyntaxError", message: "Pattern nesting exceeds maximum depth" });
      }
    });
  }

  test(`${consumer} 直接接收 AST 时仍限制遍历深度`, () => {
    for (const method of ["compile", "expand", "stringify"]) {
      let node = { type: "text", value: "a" };
      for (let depth = 0; depth < 2000; depth++) {
        const parent = { type: "paren", nodes: [node] };
        node.parent = parent;
        node = parent;
      }
      const ast = { type: "root", nodes: [node] };
      node.parent = ast;
      assert.throws(() => braces[method](ast), { name: "SyntaxError", message: "Pattern nesting exceeds maximum depth" });
    }
  });

  test(`${consumer} 不把转义字符或引号中的括号计入深度`, () => {
    assert.doesNotThrow(() => braces.compile("\\{".repeat(1000)));
    assert.doesNotThrow(() => braces.compile('"' + "{".repeat(1000) + '"'));
  });
}
