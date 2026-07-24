<template>
  <div v-loading="loading" class="app-container code-gen-sub-page">
    <el-card class="code-gen-sub-card api-doc-page__card" shadow="never">
      <div ref="swaggerRootRef" class="api-doc-page__swagger" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import SwaggerUIBundle from "swagger-ui-dist/swagger-ui-bundle.js";
import "swagger-ui-dist/swagger-ui.css";
import { getRequestAccessToken } from "@/utils/request";

const swaggerRootRef = ref<HTMLElement>();
const loading = ref(true);
const openAPIDocumentURL = "/api/docs/openapi";

/** 初始化携带当前登录令牌的 Swagger UI。 */
async function initializeSwaggerUI() {
  const swaggerRoot = swaggerRootRef.value;
  if (!swaggerRoot) return;

  SwaggerUIBundle({
    domNode: swaggerRoot,
    url: openAPIDocumentURL,
    deepLinking: true,
    displayOperationId: true,
    displayRequestDuration: true,
    docExpansion: "list",
    defaultModelsExpandDepth: -1,
    filter: true,
    onComplete: () => {
      loading.value = false;
    },
    persistAuthorization: false,
    tryItOutEnabled: true,
    validatorUrl: null,
    requestInterceptor: async request => {
      const accessToken = await getRequestAccessToken();
      if (accessToken) {
        request.headers = {
          ...request.headers,
          Authorization: accessToken
        };
      }
      return request;
    }
  });
}

onMounted(() => {
  initializeSwaggerUI();
});

onBeforeUnmount(() => {
  swaggerRootRef.value?.replaceChildren();
});
</script>

<style scoped lang="scss">
.code-gen-sub-card {
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}

.api-doc-page__card {
  flex: 1;
  min-width: 0;
  min-height: 0;
}

.api-doc-page__swagger {
  height: 100%;
  min-height: 0;
  overflow: auto;
}

:deep(.api-doc-page__card .el-card__body) {
  height: 100%;
  padding: 0;
}

:deep(.swagger-ui) {
  color: var(--el-text-color-primary);
}

:deep(.swagger-ui .information-container) {
  display: none;
}

:deep(.swagger-ui .info .title),
:deep(.swagger-ui .info p),
:deep(.swagger-ui .opblock-tag),
:deep(.swagger-ui .opblock .opblock-summary-description),
:deep(.swagger-ui .opblock .opblock-summary-path),
:deep(.swagger-ui .parameter__name),
:deep(.swagger-ui .parameter__type),
:deep(.swagger-ui table thead tr td),
:deep(.swagger-ui table thead tr th) {
  color: var(--el-text-color-primary);
}

:deep(.swagger-ui .scheme-container),
:deep(.swagger-ui .opblock .opblock-section-header),
:deep(.swagger-ui .opblock-body pre),
:deep(.swagger-ui .model-box),
:deep(.swagger-ui .dialog-ux .modal-ux) {
  background: var(--el-fill-color-light);
  box-shadow: none;
}

:deep(.swagger-ui .opblock),
:deep(.swagger-ui .opblock-tag),
:deep(.swagger-ui .model-box),
:deep(.swagger-ui .dialog-ux .modal-ux),
:deep(.swagger-ui .dialog-ux .modal-ux-content) {
  border-color: var(--el-border-color-light);
}

:deep(.swagger-ui input[type="text"]),
:deep(.swagger-ui textarea),
:deep(.swagger-ui select) {
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border-color: var(--el-border-color);
}

</style>
