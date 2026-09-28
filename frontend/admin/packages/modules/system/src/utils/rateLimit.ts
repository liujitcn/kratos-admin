import { BaseRateLimitAlgorithm } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_rate_limit_rule";

/** 限流算法的可编辑参数集合。 */
export interface RateLimitParams {
  tokens_per_second?: number;
  burst?: number;
  limit?: number;
  window_seconds?: number;
  leak_rate_per_second?: number;
  capacity?: number;
}

/** 返回所选算法的空参数表单。 */
export function defaultRateLimitParams(algorithm: BaseRateLimitAlgorithm): RateLimitParams {
  switch (algorithm) {
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_TOKEN_BUCKET:
      return { tokens_per_second: undefined, burst: undefined };
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_FIXED_WINDOW:
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_COUNTER:
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_LOG:
      return { limit: undefined, window_seconds: undefined };
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_LEAKY_BUCKET:
      return { leak_rate_per_second: undefined, capacity: undefined };
    default:
      return {};
  }
}

/** 按算法解析 JSON，只保留该算法支持的参数。 */
export function parseRateLimitParams(algorithm: BaseRateLimitAlgorithm, raw: string): RateLimitParams {
  let value: Record<string, unknown> = {};
  try {
    const parsed: unknown = JSON.parse(raw);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) value = parsed as Record<string, unknown>;
  } catch {
    return defaultRateLimitParams(algorithm);
  }
  const number = (key: string) => (typeof value[key] === "number" && Number.isFinite(value[key]) ? (value[key] as number) : undefined);
  switch (algorithm) {
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_TOKEN_BUCKET:
      return { tokens_per_second: number("tokens_per_second"), burst: number("burst") };
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_FIXED_WINDOW:
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_COUNTER:
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_LOG:
      return { limit: number("limit"), window_seconds: number("window_seconds") };
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_LEAKY_BUCKET:
      return { leak_rate_per_second: number("leak_rate_per_second"), capacity: number("capacity") };
    default:
      return defaultRateLimitParams(algorithm);
  }
}

/** 校验参数并生成仅包含当前算法字段的 JSON 对象。 */
export function serializeRateLimitParams(algorithm: BaseRateLimitAlgorithm, params: RateLimitParams): Record<string, number> | undefined {
  switch (algorithm) {
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_TOKEN_BUCKET:
      if (!isRate(params.tokens_per_second) || !isIntegerInRange(params.burst, 1, 1_000_000)) return undefined;
      return { tokens_per_second: params.tokens_per_second, burst: params.burst };
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_FIXED_WINDOW:
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_COUNTER:
      if (!isIntegerInRange(params.limit, 1, 1_000_000) || !isIntegerInRange(params.window_seconds, 1, 86_400)) return undefined;
      return { limit: params.limit, window_seconds: params.window_seconds };
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_LOG:
      if (!isIntegerInRange(params.limit, 1, 10_000) || !isIntegerInRange(params.window_seconds, 1, 86_400)) return undefined;
      return { limit: params.limit, window_seconds: params.window_seconds };
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_LEAKY_BUCKET:
      if (!isRate(params.leak_rate_per_second) || !isIntegerInRange(params.capacity, 1, 1_000_000)) return undefined;
      return { leak_rate_per_second: params.leak_rate_per_second, capacity: params.capacity };
    default:
      return undefined;
  }
}

/** 返回可直接交给国际化模板的完整参数。 */
export function rateLimitSummaryParams(algorithm: BaseRateLimitAlgorithm, raw: string): Record<string, string | number> {
  const params = parseRateLimitParams(algorithm, raw);
  return Object.fromEntries(Object.entries(params).map(([key, value]) => [key, value ?? 0])) as Record<string, string | number>;
}

/** 返回限流算法名称对应的多语言资源键。 */
export function rateLimitAlgorithmLabelKey(algorithm: BaseRateLimitAlgorithm): string {
  switch (algorithm) {
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_TOKEN_BUCKET:
      return "system.base.rate_limit_rule.algorithm.token_bucket";
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_FIXED_WINDOW:
      return "system.base.rate_limit_rule.algorithm.fixed_window";
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_COUNTER:
      return "system.base.rate_limit_rule.algorithm.sliding_window_counter";
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_LOG:
      return "system.base.rate_limit_rule.algorithm.sliding_window_log";
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_LEAKY_BUCKET:
      return "system.base.rate_limit_rule.algorithm.leaky_bucket";
    default:
      return "";
  }
}

/** 返回所选限流算法说明对应的多语言资源键。 */
export function rateLimitAlgorithmTooltipKey(algorithm: BaseRateLimitAlgorithm): string {
  switch (algorithm) {
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_TOKEN_BUCKET:
      return "system.base.rate_limit_rule.algorithm_help.token_bucket";
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_FIXED_WINDOW:
      return "system.base.rate_limit_rule.algorithm_help.fixed_window";
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_COUNTER:
      return "system.base.rate_limit_rule.algorithm_help.sliding_window_counter";
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_LOG:
      return "system.base.rate_limit_rule.algorithm_help.sliding_window_log";
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_LEAKY_BUCKET:
      return "system.base.rate_limit_rule.algorithm_help.leaky_bucket";
    default:
      return "";
  }
}

/** 返回参数摘要对应的多语言资源键。 */
export function rateLimitSummaryKey(algorithm: BaseRateLimitAlgorithm): string {
  switch (algorithm) {
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_TOKEN_BUCKET:
      return "system.base.rate_limit_rule.summary.token_bucket";
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_FIXED_WINDOW:
      return "system.base.rate_limit_rule.summary.fixed_window";
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_COUNTER:
      return "system.base.rate_limit_rule.summary.sliding_window_counter";
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_LOG:
      return "system.base.rate_limit_rule.summary.sliding_window_log";
    case BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_LEAKY_BUCKET:
      return "system.base.rate_limit_rule.summary.leaky_bucket";
    default:
      return "";
  }
}

function isRate(value: number | undefined): value is number {
  return typeof value === "number" && Number.isFinite(value) && value >= 0.001 && value <= 1_000_000;
}

function isIntegerInRange(value: number | undefined, min: number, max: number): value is number {
  return typeof value === "number" && Number.isInteger(value) && value >= min && value <= max;
}
