import { getCurrentInstance, onBeforeUnmount, ref } from "vue";

/** 语音识别结果的最小结构，便于在不同浏览器实现间复用。 */
export type SpeechResultList = ArrayLike<{
  0?: {
    transcript: string;
  };
  isFinal?: boolean;
  length: number;
}>;

/** 语音识别错误的最小结构，兼容浏览器事件和不支持错误。 */
export type SpeechRecognitionError = {
  code?: number;
  error?: string;
  message?: string;
};

type SpeechRecognitionResultEvent = {
  results: SpeechResultList;
  resultIndex: number;
};

type SpeechRecognitionInstance = {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  onstart: (() => void) | null;
  onend: (() => void) | null;
  onerror: ((error: SpeechRecognitionError) => void) | null;
  onresult: ((event: SpeechRecognitionResultEvent) => void) | null;
  start: () => void;
  stop: () => void;
};

type SpeechRecognitionConstructor = new () => SpeechRecognitionInstance;

type SpeechRecognitionWindow = Window & {
  SpeechRecognition?: SpeechRecognitionConstructor;
  webkitSpeechRecognition?: SpeechRecognitionConstructor;
};

type SpeechRecognitionOptions = {
  /** 识别错误回调。 */
  onError?: (error: SpeechRecognitionError) => void;
  /** 识别开始回调。 */
  onStart?: () => void;
  /** 识别结束回调。 */
  onEnd?: (text: string) => void;
  /** 识别结果回调。 */
  onResult?: (text: string) => void;
};

/** 增量累积语音识别结果，避免连续模式下 interim 结果重复计入已确认文本。 */
export function accumulateSpeechResultText(
  results: SpeechResultList,
  resultIndex: number,
  previousFinalText: string
): { finalText: string; text: string } {
  let finalText = previousFinalText;
  let interimText = "";
  for (let index = resultIndex; index < results.length; index++) {
    const result = results[index];
    const transcript = result?.[0]?.transcript ?? "";
    if (result?.isFinal) {
      finalText += transcript;
    } else {
      interimText += transcript;
    }
  }
  // Chrome 连续模式下 interim 结果会累计包含已确认的 final 文本，去除重复前缀。
  if (interimText && finalText && interimText.startsWith(finalText)) {
    interimText = interimText.slice(finalText.length);
  }
  return { finalText, text: finalText + interimText };
}

/** 创建浏览器语音识别实例，并统一结果、状态和卸载处理。 */
export function useSpeechRecognition({ onError, onStart, onEnd, onResult }: SpeechRecognitionOptions = {}) {
  const loading = ref(false);
  const value = ref("");
  let recognition: SpeechRecognitionInstance | null = null;
  let recognitionSession = 0;
  let finalText = "";

  /** 获取当前浏览器支持的语音识别构造函数。 */
  function resolveRecognitionConstructor() {
    if (typeof window === "undefined") return undefined;
    const speechWindow = window as SpeechRecognitionWindow;
    return speechWindow.SpeechRecognition ?? speechWindow.webkitSpeechRecognition;
  }

  /** 开始一次语音识别。 */
  function start() {
    if (loading.value) return;
    const RecognitionConstructor = resolveRecognitionConstructor();
    if (!RecognitionConstructor) {
      onError?.({ code: -1, message: "Speech recognition is not supported by this browser" });
      return;
    }

    const nextRecognition = new RecognitionConstructor();
    const currentSession = ++recognitionSession;
    recognition = nextRecognition;
    finalText = "";
    const isCurrentRecognition = () => recognition === nextRecognition && recognitionSession === currentSession;
    nextRecognition.continuous = true;
    nextRecognition.interimResults = true;
    nextRecognition.lang = "zh-CN";
    nextRecognition.onstart = () => {
      if (!isCurrentRecognition()) return;
      loading.value = true;
      value.value = "";
      onStart?.();
    };
    nextRecognition.onresult = event => {
      if (!isCurrentRecognition()) return;
      const accumulated = accumulateSpeechResultText(event.results, event.resultIndex, finalText);
      finalText = accumulated.finalText;
      value.value = accumulated.text;
      onResult?.(value.value);
    };
    nextRecognition.onerror = error => {
      if (!isCurrentRecognition()) return;
      loading.value = false;
      recognition = null;
      onError?.(error);
    };
    nextRecognition.onend = () => {
      if (!isCurrentRecognition()) return;
      loading.value = false;
      recognition = null;
      onEnd?.(value.value);
    };

    loading.value = true;
    try {
      nextRecognition.start();
    } catch (error) {
      if (!isCurrentRecognition()) return;
      loading.value = false;
      recognition = null;
      onError?.({
        code: -2,
        message: error instanceof Error ? error.message : String(error)
      });
    }
  }

  /** 停止当前语音识别。 */
  function stop() {
    const currentRecognition = recognition;
    if (!currentRecognition) return;
    recognition = null;
    recognitionSession++;
    loading.value = false;
    onEnd?.(value.value);
    currentRecognition.stop();
  }

  if (getCurrentInstance()) {
    onBeforeUnmount(() => {
      const currentRecognition = recognition;
      recognition = null;
      recognitionSession++;
      loading.value = false;
      currentRecognition?.stop();
    });
  }

  return {
    loading,
    start,
    stop,
    value
  };
}
