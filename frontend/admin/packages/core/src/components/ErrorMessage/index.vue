<script setup lang="ts" name="ErrorMessage">
import { useRouter } from "vue-router";

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
</script>

<template>
  <div class="not-container">
    <img :src="image" class="not-img" :alt="code" />
    <div class="not-detail">
      <h2>{{ code }}</h2>
      <h4>{{ message }}</h4>
      <el-button type="primary" @click="router.back()"> 返回上一页 </el-button>
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
