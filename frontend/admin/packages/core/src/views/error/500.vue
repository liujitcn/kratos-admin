<script setup lang="ts" name="500">
import errorImage from "@/assets/images/500.png";
import ErrorMessage from "@/components/ErrorMessage/index.vue";
import { useLocaleStore } from "@/locales";
const { t } = useLocaleStore();
</script>

<template>
  <ErrorMessage code="500" :image="errorImage" :message="t('core.error.500.detail')" />
</template>
