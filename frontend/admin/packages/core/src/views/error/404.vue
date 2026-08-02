<script setup lang="ts" name="404">
import errorImage from "@/assets/images/404.png";
import ErrorMessage from "@/components/ErrorMessage/index.vue";
import { useLocaleStore } from "@/locales";
const { t } = useLocaleStore();
</script>

<template>
  <ErrorMessage code="404" :image="errorImage" :message="t('core.error.404.detail')" />
</template>
