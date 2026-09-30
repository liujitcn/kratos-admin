import { defSecretCryptoService } from "@/api/base/v1/secret_crypto";
import type { GetSecretPublicKeyResponse } from "@/rpc/base/v1/secret_crypto";
import type { SecretCrypto } from "@/rpc/common/v1/types";
import type { ProFormField } from "@/components/ProForm/interface";
import { t } from "@/locales";
import * as portableCrypto from "asmcrypto.js";

type PortableCrypto = typeof import("asmcrypto.js");

/** 获取敏感字段一次性临时公钥，同一次提交的多个密文字段共用该公钥。 */
export async function fetchSecretPublicKey(): Promise<GetSecretPublicKeyResponse> {
  return defSecretCryptoService.GetSecretPublicKey({});
}

/** encodeSecretFields 在提交前把声明 secret 的字段统一加密为 SecretCrypto，返回模型浅克隆且不改动原表单。 */
export async function encodeSecretFields(
  fields: ProFormField[],
  model: Record<string, any>
): Promise<Record<string, any>> {
  const secretFields = fields.filter(field => field.secret);
  const filled = secretFields
    .map(field => ({ field, value: getByPath(model, field.prop) }))
    .filter((item): item is { field: ProFormField; value: string } => typeof item.value === "string" && item.value.trim() !== "");
  if (secretFields.length === 0) {
    return { ...model };
  }
  const publicKey = filled.length > 0 ? await fetchSecretPublicKey() : null;
  const payload = { ...model };
  for (const item of secretFields) {
    const value = getByPath(model, item.prop);
    if (typeof value === "string" && value.trim() !== "") {
      payload[item.prop] = await encryptWithPublicKey(publicKey!, value);
    } else {
      payload[item.prop] = undefined;
    }
  }
  return payload;
}

/** 按点路径读取对象字段值。 */
function getByPath(model: Record<string, any>, path: string) {
  return path.split(".").reduce((current, key) => (current == null ? current : current[key]), model as any);
}

/** 使用已获取的临时公钥加密单个敏感值，供一次取钥加密多个字段的场景复用。 */
export async function encryptWithPublicKey(publicKey: GetSecretPublicKeyResponse, value: string): Promise<SecretCrypto> {
  if (!value) {
    throw new Error(t("core.password.required"));
  }
  const cryptoApi = globalThis.crypto;
  const isSecureContext = typeof window === "undefined" || window.isSecureContext;
  if (!cryptoApi?.subtle || !isSecureContext) {
    return encryptPortableSecret(portableCrypto, value, publicKey);
  }
  const subtleCrypto = getSubtleCrypto();
  const publicKeyKey = await subtleCrypto.subtle.importKey(
    "spki",
    pemToArrayBuffer(publicKey.public_key),
    { name: "RSA-OAEP", hash: "SHA-256" },
    false,
    ["encrypt"]
  );
  const aesKey = await subtleCrypto.subtle.generateKey({ name: "AES-GCM", length: 256 }, true, ["encrypt"]);
  const rawAesKey = await subtleCrypto.subtle.exportKey("raw", aesKey);
  const iv = subtleCrypto.getRandomValues(new Uint8Array(12));
  const ciphertext = await subtleCrypto.subtle.encrypt({ name: "AES-GCM", iv }, aesKey, new TextEncoder().encode(value));
  const encryptedKey = await subtleCrypto.subtle.encrypt({ name: "RSA-OAEP" }, publicKeyKey, rawAesKey);

  return {
    key_id: publicKey.key_id,
    nonce: publicKey.nonce,
    algorithm: publicKey.algorithm,
    encrypted_key: arrayBufferToBase64(encryptedKey),
    iv: arrayBufferToBase64(iv.buffer),
    text: arrayBufferToBase64(ciphertext)
  };
}

/** 加密单个敏感字段值，自动获取一次性临时公钥。 */
export async function encryptSecret(value: string): Promise<SecretCrypto> {
  return encryptWithPublicKey(await fetchSecretPublicKey(), value);
}

/** SecretRevealHandler 按资源标识与字段名查看密钥明文的实现。 */
export type SecretRevealHandler = (resource: string, id: number, field: string) => Promise<string>;

let secretRevealHandler: SecretRevealHandler | null = null;

/** registerSecretRevealHandler 注册密钥明文查看实现，由业务模块在启动时注入。 */
export function registerSecretRevealHandler(handler: SecretRevealHandler) {
  secretRevealHandler = handler;
}

/** revealSecret 查看注册过的密钥字段明文。 */
export async function revealSecret(resource: string, id: number, field: string): Promise<string> {
  if (!secretRevealHandler) {
    throw new Error(t("core.password.reveal_unsupported"));
  }
  return secretRevealHandler(resource, id, field);
}

/** 加密密码字段：去除首尾空白且不允许为空。 */
export async function encryptPassword(value: string): Promise<SecretCrypto> {
  const plainPassword = value.trim();
  if (!plainPassword) {
    throw new Error(t("core.password.required"));
  }
  return encryptSecret(plainPassword);
}

/** 将 PEM 公钥转换为二进制 DER 数据。 */
function pemToArrayBuffer(pem: string) {
  const base64 = pem
    .replace(/-----BEGIN PUBLIC KEY-----/g, "")
    .replace(/-----END PUBLIC KEY-----/g, "")
    .replace(/\s/g, "");
  const decode = globalThis.atob;
  if (!decode) {
    throw new Error(t("core.password.base64_decode_unsupported"));
  }
  const binary = decode(base64);
  const buffer = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i += 1) {
    buffer[i] = binary.charCodeAt(i);
  }
  return buffer.buffer;
}

/** 将二进制数据编码为 base64 字符串。 */
function arrayBufferToBase64(buffer: ArrayBuffer) {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i += 1) {
    binary += String.fromCharCode(bytes[i]);
  }
  const encode = globalThis.btoa;
  if (!encode) {
    throw new Error(t("core.password.base64_encode_unsupported"));
  }
  return encode(binary);
}

/** 获取浏览器 WebCrypto 能力。 */
function getSubtleCrypto() {
  const cryptoApi = globalThis.crypto;
  const isSecureContext = typeof window === "undefined" || window.isSecureContext;
  if (!cryptoApi?.subtle || !isSecureContext) {
    throw new Error(t("core.password.crypto_unsupported"));
  }
  return cryptoApi;
}

/** 读取 DER 编码节点，用于解析 RSA SubjectPublicKeyInfo。 */
function readDerNode(data: Uint8Array, offset: number) {
  if (offset + 2 > data.length) {
    throw new Error(t("core.password.public_key_invalid"));
  }
  const tag = data[offset];
  const firstLengthByte = data[offset + 1];
  let length = firstLengthByte;
  let valueOffset = offset + 2;
  if (firstLengthByte & 0x80) {
    const lengthByteCount = firstLengthByte & 0x7f;
    if (lengthByteCount === 0 || valueOffset + lengthByteCount > data.length) {
      throw new Error(t("core.password.public_key_invalid"));
    }
    length = 0;
    for (let index = 0; index < lengthByteCount; index += 1) {
      length = length * 256 + data[valueOffset + index];
    }
    valueOffset += lengthByteCount;
  }
  const end = valueOffset + length;
  if (end > data.length) {
    throw new Error(t("core.password.public_key_invalid"));
  }
  return { tag, value: data.subarray(valueOffset, end), next: end };
}

/** 去除 DER 正整数前导零字节。 */
function trimDerInteger(value: Uint8Array) {
  let offset = 0;
  while (offset < value.length - 1 && value[offset] === 0) {
    offset += 1;
  }
  return value.subarray(offset);
}

/** 从 PEM SubjectPublicKeyInfo 中提取 RSA 模数和指数。 */
function parsePortableRsaPublicKey(publicKey: string): [Uint8Array, Uint8Array] {
  const der = new Uint8Array(pemToArrayBuffer(publicKey));
  const subjectPublicKeyInfo = readDerNode(der, 0);
  const algorithm = readDerNode(subjectPublicKeyInfo.value, 0);
  const bitString = readDerNode(subjectPublicKeyInfo.value, algorithm.next);
  if (bitString.tag !== 0x03 || bitString.value[0] !== 0) {
    throw new Error(t("core.password.public_key_invalid"));
  }
  const rsaPublicKey = readDerNode(bitString.value, 1);
  const modulus = readDerNode(rsaPublicKey.value, 0);
  const exponent = readDerNode(rsaPublicKey.value, modulus.next);
  if (modulus.tag !== 0x02 || exponent.tag !== 0x02) {
    throw new Error(t("core.password.public_key_invalid"));
  }
  return [trimDerInteger(modulus.value), trimDerInteger(exponent.value)];
}

/** 计算 RSA-OAEP 所需的 MGF1 掩码。 */
function generateMgf1Mask(cryptoApi: PortableCrypto, seed: Uint8Array, length: number) {
  const mask = new Uint8Array(length);
  const counter = new Uint8Array(4);
  const hash = new cryptoApi.Sha256();
  const blockCount = Math.ceil(length / hash.HASH_SIZE);
  for (let index = 0; index < blockCount; index += 1) {
    counter[0] = index >>> 24;
    counter[1] = index >>> 16;
    counter[2] = index >>> 8;
    counter[3] = index;
    let block = hash.reset().process(seed).process(counter).finish().result;
    if (!block) {
      throw new Error(t("core.password.encrypt_failed"));
    }
    const target = mask.subarray(index * hash.HASH_SIZE);
    if (block.length > target.length) {
      block = block.subarray(0, target.length);
    }
    target.set(block);
  }
  return mask;
}

/** 生成纯 JavaScript 加密所需的随机字节。 */
function getPortableRandomBytes(length: number) {
  const cryptoApi = globalThis.crypto;
  if (!cryptoApi?.getRandomValues) {
    throw new Error(t("core.password.random_failed"));
  }
  return cryptoApi.getRandomValues(new Uint8Array(length));
}

/** 使用纯 JavaScript 实现兼容缺少 WebCrypto 环境的敏感值加密协议。 */
async function encryptPortableSecret(
  cryptoApi: PortableCrypto,
  value: string,
  publicKeyResponse: GetSecretPublicKeyResponse
): Promise<SecretCrypto> {
  const aesKey = getPortableRandomBytes(32);
  const iv = getPortableRandomBytes(12);
  const plaintext = cryptoApi.string_to_bytes(value, true);
  const [modulus, exponent] = parsePortableRsaPublicKey(publicKeyResponse.public_key);
  const hash = new cryptoApi.Sha256();
  const keySize = Math.ceil(new cryptoApi.BigNumber(modulus).bitLength / 8);
  const seed = getPortableRandomBytes(hash.HASH_SIZE);
  const dataBlockLength = keySize - hash.HASH_SIZE - 1;
  const paddingLength = dataBlockLength - aesKey.length - hash.HASH_SIZE - 1;
  if (paddingLength < 0) {
    throw new Error(t("core.password.password_too_long"));
  }
  const encodedMessage = new Uint8Array(keySize);
  const maskedSeed = encodedMessage.subarray(1, hash.HASH_SIZE + 1);
  const dataBlock = encodedMessage.subarray(hash.HASH_SIZE + 1);
  const labelHash = hash.reset().process(new Uint8Array()).finish().result;
  if (!labelHash) {
    throw new Error(t("core.password.encrypt_failed"));
  }
  dataBlock.set(labelHash, 0);
  dataBlock.set(aesKey, hash.HASH_SIZE + paddingLength + 1);
  dataBlock[hash.HASH_SIZE + paddingLength] = 1;
  const dataBlockMask = generateMgf1Mask(cryptoApi, seed, dataBlock.length);
  for (let index = 0; index < dataBlock.length; index += 1) {
    dataBlock[index] ^= dataBlockMask[index];
  }
  const seedMask = generateMgf1Mask(cryptoApi, dataBlock, maskedSeed.length);
  maskedSeed.set(seed);
  for (let index = 0; index < maskedSeed.length; index += 1) {
    maskedSeed[index] ^= seedMask[index];
  }
  const encryptedKey = new cryptoApi.RSA([modulus, exponent]).encrypt(new cryptoApi.BigNumber(encodedMessage)).result;
  if (!encryptedKey) {
    throw new Error(t("core.password.encrypt_failed"));
  }
  const ciphertext = cryptoApi.AES_GCM.encrypt(plaintext, aesKey, iv);

  return {
    key_id: publicKeyResponse.key_id,
    nonce: publicKeyResponse.nonce,
    algorithm: publicKeyResponse.algorithm,
    encrypted_key: arrayBufferToBase64(encryptedKey.slice().buffer),
    iv: arrayBufferToBase64(iv.buffer),
    text: arrayBufferToBase64(ciphertext.slice().buffer)
  };
}
