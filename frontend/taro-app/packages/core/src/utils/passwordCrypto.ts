import Taro from '@tarojs/taro'
import * as miniCrypto from 'asmcrypto.js'
import { defLoginService } from '../api/base/login'
import { t } from '../locales'
import { PasswordCryptoScene } from '../rpc/base/v1/login'
import type { PasswordCrypto } from '../rpc/common/v1/types'

type MiniCrypto = typeof import('asmcrypto.js')

export const PASSWORD_CRYPTO_SCENE = PasswordCryptoScene
export type { PasswordCryptoScene }

/** 将 PEM 公钥转换为二进制 DER 数据。 */
function pemToArrayBuffer(pem: string) {
  const base64 = pem
    .replace(/-----BEGIN PUBLIC KEY-----/g, '')
    .replace(/-----END PUBLIC KEY-----/g, '')
    .replace(/\s/g, '')
  const decode = globalThis.atob
  if (!decode) {
    throw new Error(t('core.crypto.base64_decode_unsupported'))
  }
  const binary = decode(base64)
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
  const encode = globalThis.btoa
  if (!encode) {
    throw new Error(t('core.crypto.base64_encode_unsupported'))
  }
  return encode(binary)
}

/** 生成纯 JavaScript 密码加密所需的随机字节。 */
function getRandomBytes(length: number) {
  const cryptoApi = globalThis.crypto
  if (cryptoApi?.getRandomValues) {
    return Promise.resolve(cryptoApi.getRandomValues(new Uint8Array(length)).buffer)
  }
  return Taro.getRandomValues({ length }).then(({ randomValues }) => randomValues)
}

type DerNode = {
  tag: number
  value: Uint8Array
  next: number
}

/** 读取 DER 编码节点，用于解析 RSA SubjectPublicKeyInfo。 */
function readDerNode(data: Uint8Array, offset: number): DerNode {
  if (offset + 2 > data.length) {
    throw new Error(t('core.crypto.public_key_invalid'))
  }
  const tag = data[offset]
  const firstLengthByte = data[offset + 1]
  let length = firstLengthByte
  let valueOffset = offset + 2
  if (firstLengthByte & 0x80) {
    const lengthByteCount = firstLengthByte & 0x7f
    if (lengthByteCount === 0 || valueOffset + lengthByteCount > data.length) {
      throw new Error(t('core.crypto.public_key_invalid'))
    }
    length = 0
    for (let index = 0; index < lengthByteCount; index += 1) {
      length = length * 256 + data[valueOffset + index]
    }
    valueOffset += lengthByteCount
  }
  const end = valueOffset + length
  if (end > data.length) {
    throw new Error(t('core.crypto.public_key_invalid'))
  }
  return {
    tag,
    value: data.subarray(valueOffset, end),
    next: end,
  }
}

/** 去除 DER 正整数前导零字节。 */
function trimDerInteger(value: Uint8Array) {
  let offset = 0
  while (offset < value.length - 1 && value[offset] === 0) {
    offset += 1
  }
  return value.subarray(offset)
}

/** 从 PEM SubjectPublicKeyInfo 中提取 RSA 模数和指数。 */
function parseMiniRsaPublicKey(publicKey: string): [Uint8Array, Uint8Array] {
  const base64 = publicKey
    .replace(/-----BEGIN PUBLIC KEY-----/g, '')
    .replace(/-----END PUBLIC KEY-----/g, '')
    .replace(/\s/g, '')
  const der = new Uint8Array(Taro.base64ToArrayBuffer(base64))
  const subjectPublicKeyInfo = readDerNode(der, 0)
  const algorithm = readDerNode(subjectPublicKeyInfo.value, 0)
  const bitString = readDerNode(subjectPublicKeyInfo.value, algorithm.next)
  if (bitString.tag !== 0x03 || bitString.value[0] !== 0) {
    throw new Error(t('core.crypto.public_key_invalid'))
  }
  const rsaPublicKey = readDerNode(bitString.value, 1)
  const modulus = readDerNode(rsaPublicKey.value, 0)
  const exponent = readDerNode(rsaPublicKey.value, modulus.next)
  if (modulus.tag !== 0x02 || exponent.tag !== 0x02) {
    throw new Error(t('core.crypto.public_key_invalid'))
  }
  return [trimDerInteger(modulus.value), trimDerInteger(exponent.value)]
}

/** 计算 RSA-OAEP 所需的 MGF1 掩码。 */
function generateMgf1Mask(crypto: MiniCrypto, seed: Uint8Array, length: number) {
  const mask = new Uint8Array(length)
  const counter = new Uint8Array(4)
  const hash = new crypto.Sha256()
  const blockCount = Math.ceil(length / hash.HASH_SIZE)
  for (let index = 0; index < blockCount; index += 1) {
    counter[0] = index >>> 24
    counter[1] = index >>> 16
    counter[2] = index >>> 8
    counter[3] = index
    let block = hash.reset().process(seed).process(counter).finish().result
    if (!block) {
      throw new Error(t('core.crypto.encrypt_failed'))
    }
    const target = mask.subarray(index * hash.HASH_SIZE)
    if (block.length > target.length) {
      block = block.subarray(0, target.length)
    }
    target.set(block)
  }
  return mask
}

/** 将字节数组编码为当前运行端可提交的 Base64。 */
function miniBytesToBase64(value: Uint8Array) {
  const buffer = value.slice().buffer
  return process.env.TARO_ENV === 'h5'
    ? arrayBufferToBase64(buffer)
    : Taro.arrayBufferToBase64(buffer)
}

/** 使用纯 JavaScript 实现兼容缺少 WebCrypto 的运行时。 */
async function encryptPortablePassword(
  crypto: MiniCrypto,
  password: string,
  publicKeyResponse: {
    key_id: string
    public_key: string
    algorithm: string
    nonce: string
  },
): Promise<PasswordCrypto> {
  const aesKey = new Uint8Array(await getRandomBytes(32))
  const iv = new Uint8Array(await getRandomBytes(12))
  const plaintext = crypto.string_to_bytes(password, true)
  const [modulus, exponent] = parseMiniRsaPublicKey(publicKeyResponse.public_key)
  const hash = new crypto.Sha256()
  const keySize = Math.ceil(new crypto.BigNumber(modulus).bitLength / 8)
  const seed = new Uint8Array(await getRandomBytes(hash.HASH_SIZE))
  const dataBlockLength = keySize - hash.HASH_SIZE - 1
  const paddingLength = dataBlockLength - aesKey.length - hash.HASH_SIZE - 1
  if (paddingLength < 0) {
    throw new Error(t('core.crypto.password_too_long'))
  }
  const encodedMessage = new Uint8Array(keySize)
  const maskedSeed = encodedMessage.subarray(1, hash.HASH_SIZE + 1)
  const dataBlock = encodedMessage.subarray(hash.HASH_SIZE + 1)
  const labelHash = hash.reset().process(new Uint8Array()).finish().result
  if (!labelHash) {
    throw new Error(t('core.crypto.encrypt_failed'))
  }
  dataBlock.set(labelHash, 0)
  dataBlock.set(aesKey, hash.HASH_SIZE + paddingLength + 1)
  dataBlock[hash.HASH_SIZE + paddingLength] = 1
  const dataBlockMask = generateMgf1Mask(crypto, seed, dataBlock.length)
  for (let index = 0; index < dataBlock.length; index += 1) {
    dataBlock[index] ^= dataBlockMask[index]
  }
  const seedMask = generateMgf1Mask(crypto, dataBlock, maskedSeed.length)
  maskedSeed.set(seed)
  for (let index = 0; index < maskedSeed.length; index += 1) {
    maskedSeed[index] ^= seedMask[index]
  }
  const encryptedKey = new crypto.RSA([modulus, exponent]).encrypt(
    new crypto.BigNumber(encodedMessage),
  ).result
  if (!encryptedKey) {
    throw new Error(t('core.crypto.encrypt_failed'))
  }
  const ciphertext = crypto.AES_GCM.encrypt(plaintext, aesKey, iv)
  return {
    key_id: publicKeyResponse.key_id,
    nonce: publicKeyResponse.nonce,
    algorithm: publicKeyResponse.algorithm,
    encrypted_key: miniBytesToBase64(encryptedKey),
    iv: miniBytesToBase64(iv),
    ciphertext: miniBytesToBase64(ciphertext),
  }
}

/** 加密单个密码字段，返回后端可解析的密码密文。 */
export async function encryptPassword(
  password: string,
  scene: PasswordCryptoScene,
): Promise<PasswordCrypto> {
  const plainPassword = password.trim()
  if (!plainPassword) {
    throw new Error(t('core.crypto.password_required'))
  }

  const publicKeyResponse = await defLoginService.PasswordPublicKey({ scene })

  const cryptoApi = globalThis.crypto
  // 微信小程序和局域网 HTTP 页面没有 WebCrypto，使用纯 JavaScript 实现保持协议一致。
  if (process.env.TARO_ENV === 'weapp' || !cryptoApi?.subtle) {
    return encryptPortablePassword(miniCrypto, plainPassword, publicKeyResponse)
  }

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
