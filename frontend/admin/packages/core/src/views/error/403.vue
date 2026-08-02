<script setup lang="ts" name="403">
import errorImage from "@/assets/images/403.png";
import ErrorMessage from "@/components/ErrorMessage/index.vue";
import { useLocaleStore } from "@/locales";
const { t } = useLocaleStore();
</script>

<template>
  <ErrorMessage code="403" :image="errorImage" :message="t('core.error.403.detail')" />
</template>
