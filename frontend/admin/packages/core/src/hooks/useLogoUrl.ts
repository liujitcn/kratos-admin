import type { Ref } from "vue";
import defaultLogoUrl from "@/assets/images/logo.svg";
import { useImageUrl } from "@/hooks/useImageUrl";

/**
 * 管理站点 Logo 地址，并在配置地址不可用时回退到本地默认 Logo。
 */
export function useLogoUrl(configuredLogoUrl: Readonly<Ref<string>>) {
  const { imageUrl, handleImageError } = useImageUrl(configuredLogoUrl, defaultLogoUrl);
  return { logoUrl: imageUrl, handleLogoError: handleImageError };
}
