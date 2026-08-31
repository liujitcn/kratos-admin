import { t } from "@/locales";

/** WebAuthn 选项使用的 Base64URL 编解码。 */
function decodeBase64Url(value: string): ArrayBuffer {
  const normalized = value.replace(/-/g, "+").replace(/_/g, "/");
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=");
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) bytes[index] = binary.charCodeAt(index);
  return bytes.buffer;
}

/** 将 WebAuthn 二进制响应编码为服务端协议使用的 Base64URL。 */
function encodeBase64Url(value: ArrayBuffer | Uint8Array): string {
  const bytes = value instanceof Uint8Array ? value : new Uint8Array(value);
  let binary = "";
  bytes.forEach(byte => {
    binary += String.fromCharCode(byte);
  });
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

/** 将服务端返回的 WebAuthn 创建选项转换为浏览器类型。 */
function parseCreationOptions(optionsJson: string): PublicKeyCredentialCreationOptions {
  const options = JSON.parse(optionsJson) as { publicKey: PublicKeyCredentialCreationOptions };
  const publicKey = options.publicKey;
  publicKey.challenge = decodeBase64Url(publicKey.challenge as unknown as string);
  publicKey.user.id = decodeBase64Url(publicKey.user.id as unknown as string);
  publicKey.excludeCredentials = publicKey.excludeCredentials?.map(item => ({
    ...item,
    id: decodeBase64Url(item.id as unknown as string)
  }));
  return publicKey;
}

/** 将服务端返回的 WebAuthn 断言选项转换为浏览器类型。 */
function parseRequestOptions(optionsJson: string): PublicKeyCredentialRequestOptions {
  const options = JSON.parse(optionsJson) as { publicKey: PublicKeyCredentialRequestOptions };
  const publicKey = options.publicKey;
  publicKey.challenge = decodeBase64Url(publicKey.challenge as unknown as string);
  publicKey.allowCredentials = publicKey.allowCredentials?.map(item => ({
    ...item,
    id: decodeBase64Url(item.id as unknown as string)
  }));
  return publicKey;
}

/** 将 PublicKeyCredential 序列化为可提交给服务端的 JSON。 */
function serializeCredential(credential: PublicKeyCredential): string {
  const response = credential.response;
  const serializedResponse: Record<string, unknown> = {
    clientDataJSON: encodeBase64Url(response.clientDataJSON)
  };
  const payload: Record<string, unknown> = {
    id: credential.id,
    rawId: encodeBase64Url(credential.rawId),
    type: credential.type,
    response: serializedResponse,
    clientExtensionResults: credential.getClientExtensionResults()
  };
  if ("attestationObject" in response) {
    const attestation = response as AuthenticatorAttestationResponse;
    serializedResponse.attestationObject = encodeBase64Url(attestation.attestationObject);
  } else {
    const assertion = response as AuthenticatorAssertionResponse;
    Object.assign(serializedResponse, {
      authenticatorData: encodeBase64Url(assertion.authenticatorData),
      signature: encodeBase64Url(assertion.signature),
      userHandle: assertion.userHandle ? encodeBase64Url(assertion.userHandle) : null
    });
  }
  if (credential.authenticatorAttachment) payload.authenticatorAttachment = credential.authenticatorAttachment;
  return JSON.stringify(payload);
}

/** 执行 WebAuthn 注册 ceremony。 */
export async function createWebAuthnCredential(optionsJson: string): Promise<string> {
  if (!navigator.credentials?.create) throw new Error(t("core.login.mfa_webauthn_setup_unsupported"));
  const credential = await navigator.credentials.create({ publicKey: parseCreationOptions(optionsJson) });
  if (!(credential instanceof PublicKeyCredential)) throw new Error(t("core.login.mfa_webauthn_setup_invalid"));
  return serializeCredential(credential);
}

/** 执行 WebAuthn 登录 ceremony。 */
export async function getWebAuthnAssertion(optionsJson: string): Promise<string> {
  if (!navigator.credentials?.get) throw new Error(t("core.login.mfa_webauthn_login_unsupported"));
  const credential = await navigator.credentials.get({ publicKey: parseRequestOptions(optionsJson) });
  if (!(credential instanceof PublicKeyCredential)) throw new Error(t("core.login.mfa_webauthn_login_invalid"));
  return serializeCredential(credential);
}
