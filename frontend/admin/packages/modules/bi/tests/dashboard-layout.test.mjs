import assert from "node:assert/strict";
import { test } from "node:test";
import { flowPlace } from "../src/schema/layout.ts";

test("组装卡片的布局无覆盖且不修改源引用", () => {
  const cards = [
    { id: 1, w: 6, h: 4 },
    { id: 2, w: 6, h: 4 },
    { id: 3, w: 12, h: 3 }
  ];
  const result = flowPlace(cards, 12);
  assert.deepEqual(
    result.map(({ x, y }) => [x, y]),
    [
      [0, 0],
      [6, 0],
      [0, 4]
    ]
  );
  assert.equal(cards[0].x, undefined);
  assert.throws(() => flowPlace([{ w: 13, h: 4 }], 12));
  assert.throws(() => flowPlace([{ w: 6, h: 0 }], 12));
});
