declare module "swagger-ui-dist/swagger-ui-bundle.js" {
  /** Swagger UI 发起请求时使用的配置。 */
  interface SwaggerUIRequest {
    headers?: Record<string, string>;
  }

  /** Swagger UI 初始化配置。 */
  interface SwaggerUIConfig {
    defaultModelsExpandDepth?: number;
    deepLinking?: boolean;
    displayOperationId?: boolean;
    displayRequestDuration?: boolean;
    docExpansion?: "full" | "list" | "none";
    domNode: HTMLElement;
    filter?: boolean;
    onComplete?: () => void;
    persistAuthorization?: boolean;
    requestInterceptor?: (request: SwaggerUIRequest) => Promise<SwaggerUIRequest>;
    tryItOutEnabled?: boolean;
    url: string;
    validatorUrl?: null;
  }

  /** SwaggerUIBundle 创建 API 文档界面。 */
  function SwaggerUIBundle(config: SwaggerUIConfig): unknown;

  export default SwaggerUIBundle;
}
