<script setup lang="ts" name="403">
import errorImage from "@/assets/images/403.png";
import ErrorMessage from "@/components/ErrorMessage/index.vue";
</script>

<template>
  <ErrorMessage code="403" :image="errorImage" message="抱歉，您无权访问该页面~🙅‍♂️🙅‍♀️" />
</template>
