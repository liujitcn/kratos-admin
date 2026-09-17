import assert from "node:assert/strict";
import test from "node:test";
import { useSpeechRecognition } from "../src/views/ai/chat/components/speech-recognition.js";
import { collectSpeechResultText } from "../src/views/ai/chat/components/speech-recognition.js";

test("语音输入应保留同一次识别事件中的全部结果", () => {
  const results = [
    { 0: { transcript: "第一句" }, length: 1 },
    { 0: { transcript: "第二句" }, length: 1 }
  ];

  assert.equal(collectSpeechResultText(results), "第一句第二句");
});

test("点击停止后应立即退出录音态，不等待浏览器 onend", () => {
  const previousWindow = globalThis.window;
  let endCount = 0;

  class FakeSpeechRecognition {
    static active: FakeSpeechRecognition | undefined;
    onstart?: () => void;
    onend?: () => void;
    onerror?: () => void;
    onresult?: () => void;

    start() {
      FakeSpeechRecognition.active = this;
      this.onstart?.();
    }

    stop() {
      // 模拟浏览器延迟触发 onend。
    }
  }

  Object.assign(globalThis, {
    window: { SpeechRecognition: FakeSpeechRecognition }
  });

  try {
    const recognition = useSpeechRecognition({
      onEnd: () => {
        endCount++;
      }
    });

    recognition.start();
    assert.equal(recognition.loading.value, true);
    recognition.stop();
    const activeRecognition = FakeSpeechRecognition.active;

    assert.equal(recognition.loading.value, false);
    assert.equal(endCount, 1);
    assert.ok(activeRecognition);
    activeRecognition.onend?.();
    assert.equal(endCount, 1);
  } finally {
    Object.assign(globalThis, { window: previousWindow });
  }
});
