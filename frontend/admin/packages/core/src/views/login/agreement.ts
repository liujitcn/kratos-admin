import { computed, ref, watch } from "vue";
import type { AdminAgreementConfig } from "@/stores/interface";

/** 管理端协议阅读类型。 */
export type AdminAgreementKind = "service" | "privacy";

/** 保持管理端协议默认未选，并在正文或版本更新后要求重新主动确认。 */
export function useAdminAgreementConsent(agreements: () => AdminAgreementConfig) {
  const accepted = ref(false);
  const ready = computed(() => {
    const policy = agreements();
    return Boolean(policy.service && policy.serviceVersion && policy.privacy && policy.privacyVersion);
  });
  const confirmation = computed(() => {
    if (!accepted.value || !ready.value) return undefined;
    return {
      accepted: true,
      service_version: agreements().serviceVersion,
      privacy_version: agreements().privacyVersion
    };
  });

  watch(
    () => [agreements().service, agreements().serviceVersion, agreements().privacy, agreements().privacyVersion],
    () => {
      accepted.value = false;
    },
    { flush: "sync" }
  );

  return { accepted, ready, confirmation };
}
