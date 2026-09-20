import { ref, watch, type Ref } from "vue";

/**
 * 管理图片地址，并在配置地址为空或加载失败时回退到指定的本地资源。
 */
export function useImageUrl(configuredImageUrl: Readonly<Ref<string>>, fallbackImageUrl: string) {
  const imageUrl = ref(fallbackImageUrl);

  watch(
    configuredImageUrl,
    value => {
      imageUrl.value = value || fallbackImageUrl;
    },
    { immediate: true }
  );

  /** 图片加载失败时回退到指定的本地资源。 */
  const handleImageError = () => {
    if (imageUrl.value !== fallbackImageUrl) {
      imageUrl.value = fallbackImageUrl;
    }
  };

  return { imageUrl, handleImageError };
}
