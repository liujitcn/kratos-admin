<script setup lang="ts" name="ErrorMessage">
import { useRouter } from "vue-router";
import { useLocaleStore } from "@/locales";

/** 错误状态展示属性。 */
interface ErrorMessageProps {
  /** HTTP 状态码。 */
  code: string;
  /** 状态插图地址。 */
  image: string;
  /** 用户提示文案。 */
  message: string;
}

defineProps<ErrorMessageProps>();

const router = useRouter();
const { t } = useLocaleStore();
</script>

<template>
  <div class="not-container">
    <el-image :src="image" class="not-img" :alt="code" />
    <div class="not-detail">
      <el-text tag="h2">{{ code }}</el-text>
      <el-text tag="h4">{{ message }}</el-text>
      <el-button type="primary" @click="router.back()">{{ t("common.action.back") }}</el-button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.not-container {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  .not-img {
    margin-right: 120px;
    // el-image 容器不定尺寸，由图片固有尺寸撑开
    :deep(img) {
      width: auto;
      height: auto;
    }
  }
  .not-detail {
    display: flex;
    flex-direction: column;
    h2,
    h4 {
      padding: 0;
      margin: 0;
    }
    h2 {
      font-size: 60px;
      color: var(--el-text-color-primary);
    }
    h4 {
      margin: 30px 0 20px;
      font-size: 19px;
      font-weight: normal;
      color: var(--el-text-color-regular);
    }
    .el-button {
      width: 100px;
    }
  }
}
</style>
