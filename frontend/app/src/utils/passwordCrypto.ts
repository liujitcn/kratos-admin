import { defLoginService } from '@/api/base/login'
import { PasswordCryptoScene } from '@/rpc/common/v1/enum'
import type { PasswordCrypto } from '@/rpc/common/v1/types'

export const PASSWORD_CRYPTO_SCENE = PasswordCryptoScene
export type { PasswordCryptoScene }

/** 将 PEM 公钥转换为二进制 DER 数据。 */
function pemToArrayBuffer(pem: string) {
  const base64 = pem
    .replace(/-----BEGIN PUBLIC KEY-----/g, '')
    .replace(/-----END PUBLIC KEY-----/g, '')
    .replace(/\s/g, '')
  const binary = window.atob(base64)
  const buffer = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i += 1) {
    buffer[i] = binary.charCodeAt(i)
  }
  return buffer.buffer
}

/** 将二进制数据编码为 base64 字符串。 */
function arrayBufferToBase64(buffer: ArrayBuffer) {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (let i = 0; i < bytes.byteLength; i += 1) {
    binary += String.fromCharCode(bytes[i])
  }
  return window.btoa(binary)
}

/** 获取浏览器 WebCrypto 能力。 */
function getSubtleCrypto() {
  const cryptoApi = globalThis.crypto
  if (!cryptoApi?.subtle) {
    throw new Error('当前环境不支持密码加密')
  }
  return cryptoApi
}

/** 加密单个密码字段，返回后端可解析的密码密文。 */
export async function encryptPassword(
  password: string,
  scene: PasswordCryptoScene,
): Promise<PasswordCrypto> {
  const plainPassword = password.trim()
  if (!plainPassword) {
    throw new Error('密码不能为空')
  }

  const cryptoApi = getSubtleCrypto()
  const publicKeyResponse = await defLoginService.PasswordPublicKey({ scene })
  const publicKey = await cryptoApi.subtle.importKey(
    'spki',
    pemToArrayBuffer(publicKeyResponse.public_key),
    { name: 'RSA-OAEP', hash: 'SHA-256' },
    false,
    ['encrypt'],
  )
  const aesKey = await cryptoApi.subtle.generateKey({ name: 'AES-GCM', length: 256 }, true, [
    'encrypt',
  ])
  const rawAesKey = await cryptoApi.subtle.exportKey('raw', aesKey)
  const iv = cryptoApi.getRandomValues(new Uint8Array(12))
  const ciphertext = await cryptoApi.subtle.encrypt(
    { name: 'AES-GCM', iv },
    aesKey,
    new TextEncoder().encode(plainPassword),
  )
  const encryptedKey = await cryptoApi.subtle.encrypt({ name: 'RSA-OAEP' }, publicKey, rawAesKey)

  return {
    key_id: publicKeyResponse.key_id,
    nonce: publicKeyResponse.nonce,
    algorithm: publicKeyResponse.algorithm,
    encrypted_key: arrayBufferToBase64(encryptedKey),
    iv: arrayBufferToBase64(iv.buffer),
    ciphertext: arrayBufferToBase64(ciphertext),
  }
}
