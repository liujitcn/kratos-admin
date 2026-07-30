<script setup lang="ts" name="404">
import errorImage from "@/assets/images/404.png";
import ErrorMessage from "@/components/ErrorMessage/index.vue";
</script>

<template>
  <ErrorMessage code="404" :image="errorImage" message="抱歉，您访问的页面不存在~🤷‍♂️🤷‍♀️" />
</template>
