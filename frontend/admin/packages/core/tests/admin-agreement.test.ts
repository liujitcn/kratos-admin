import assert from "node:assert/strict";
import test from "node:test";
import { reactive } from "vue";
import { useAdminAgreementConsent } from "../src/views/login/agreement.js";

test("管理端登录默认不勾选协议，主动勾选后仅提交管理端版本", () => {
  const agreements = reactive({
    service: "管理端服务正文",
    serviceVersion: "admin-service-v1",
    privacy: "管理端隐私正文",
    privacyVersion: "admin-privacy-v1"
  });
  const consent = useAdminAgreementConsent(() => agreements);
  assert.equal(consent.accepted.value, false);
  assert.equal(consent.confirmation.value, undefined);

  consent.accepted.value = true;
  assert.deepEqual(consent.confirmation.value, {
    accepted: true,
    service_version: "admin-service-v1",
    privacy_version: "admin-privacy-v1"
  });
});

test("管理端协议正文或版本改变立即清除已有勾选", () => {
  const agreements = reactive({ service: "服务正文", serviceVersion: "v1", privacy: "隐私正文", privacyVersion: "v1" });
  const consent = useAdminAgreementConsent(() => agreements);
  consent.accepted.value = true;
  agreements.serviceVersion = "v2";
  assert.equal(consent.accepted.value, false);
  assert.equal(consent.confirmation.value, undefined);

  consent.accepted.value = true;
  agreements.privacy = "已更新的隐私正文";
  assert.equal(consent.accepted.value, false);
});

test("缺少管理端专属正文或版本时不能同意登录", () => {
  const agreements = reactive({ service: "", serviceVersion: "", privacy: "", privacyVersion: "" });
  const consent = useAdminAgreementConsent(() => agreements);
  consent.accepted.value = true;
  assert.equal(consent.ready.value, false);
  assert.equal(consent.confirmation.value, undefined);
});
