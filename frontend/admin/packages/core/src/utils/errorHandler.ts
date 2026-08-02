import { ElNotification } from "element-plus";
import { t } from "@/locales";

/**
 * @description 全局代码错误捕捉
 * */
const errorHandler = (error: any) => {
  // 过滤 HTTP 请求错误
  if (error.status || error.status == 0) return false;
  let errorMap: { [key: string]: string } = {
    InternalError: t("core.error.javascript.internal"),
    ReferenceError: t("core.error.javascript.reference"),
    TypeError: t("core.error.javascript.type"),
    RangeError: t("core.error.javascript.range"),
    SyntaxError: t("core.error.javascript.syntax"),
    EvalError: t("core.error.javascript.eval"),
    URIError: t("core.error.javascript.uri")
  };
  let errorName = errorMap[error.name] || t("core.error.javascript.unknown");
  ElNotification({
    title: errorName,
    message: error,
    type: "error",
    duration: 3000
  });
};

export default errorHandler;
