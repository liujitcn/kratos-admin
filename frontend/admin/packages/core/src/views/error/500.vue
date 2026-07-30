<script setup lang="ts" name="500">
import errorImage from "@/assets/images/500.png";
import ErrorMessage from "@/components/ErrorMessage/index.vue";
</script>

<template>
  <ErrorMessage code="500" :image="errorImage" message="抱歉，您的网络不见了~🤦‍♂️🤦‍♀️" />
</template>
