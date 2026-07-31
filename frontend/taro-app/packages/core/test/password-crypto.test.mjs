import assert from 'node:assert/strict'
import { Buffer } from 'node:buffer'
import { resolve } from 'node:path'
import test from 'node:test'
import { build } from 'esbuild'

test('encryptPassword falls back when WebCrypto subtle is unavailable', async () => {
  const coreRoot = resolve(import.meta.dirname, '..')
  const result = await build({
    stdin: {
      contents: `
        import {
          constants,
          createDecipheriv,
          generateKeyPairSync,
          privateDecrypt,
          webcrypto,
        } from 'node:crypto'
        import { defLoginService } from './src/api/base/login.ts'
        import { encryptPassword, PASSWORD_CRYPTO_SCENE } from './src/utils/passwordCrypto.ts'

        const { publicKey, privateKey } = generateKeyPairSync('rsa', { modulusLength: 2048 })
        const originalCrypto = globalThis.crypto
        Object.defineProperty(globalThis, 'crypto', {
          configurable: true,
          value: { getRandomValues: webcrypto.getRandomValues.bind(webcrypto) },
        })
        defLoginService.PasswordPublicKey = async () => ({
          key_id: 'fallback-test-key',
          public_key: publicKey.export({ type: 'spki', format: 'pem' }).toString(),
          algorithm: 'RSA-OAEP-256+A256GCM',
          nonce: 'fallback-test-nonce',
          expires_in: 300,
        })

        let encrypted
        try {
          encrypted = await encryptPassword('112233', PASSWORD_CRYPTO_SCENE.LOGIN)
        } finally {
          Object.defineProperty(globalThis, 'crypto', {
            configurable: true,
            value: originalCrypto,
          })
        }

        const aesKey = privateDecrypt(
          {
            key: privateKey,
            oaepHash: 'sha256',
            padding: constants.RSA_PKCS1_OAEP_PADDING,
          },
          Buffer.from(encrypted.encrypted_key, 'base64'),
        )
        const encryptedPayload = Buffer.from(encrypted.ciphertext, 'base64')
        const decipher = createDecipheriv(
          'aes-256-gcm',
          aesKey,
          Buffer.from(encrypted.iv, 'base64'),
        )
        decipher.setAuthTag(encryptedPayload.subarray(-16))
        export const plaintext = Buffer.concat([
          decipher.update(encryptedPayload.subarray(0, -16)),
          decipher.final(),
        ]).toString()
      `,
      loader: 'ts',
      resolveDir: coreRoot,
      sourcefile: 'password-crypto-fallback.ts',
    },
    bundle: true,
    define: { 'process.env.TARO_ENV': JSON.stringify('h5') },
    format: 'esm',
    platform: 'node',
    plugins: [
      {
        name: 'taro-stub',
        setup(context) {
          context.onResolve({ filter: /^@tarojs\/taro$/ }, () => ({
            namespace: 'taro-stub',
            path: 'taro-stub',
          }))
          context.onLoad({ filter: /.*/, namespace: 'taro-stub' }, () => ({
            contents: `
              export const getCurrentPages = () => []
              export default {
                arrayBufferToBase64(value) {
                  // Taro H5 4.2.1 passes ArrayBuffer to base64-js, which returns an empty string.
                  return typeof value.length === 'number'
                    ? Buffer.from(value).toString('base64')
                    : ''
                },
                base64ToArrayBuffer(value) {
                  return Uint8Array.from(Buffer.from(value, 'base64')).buffer
                },
                getRandomValues({ length }) {
                  const values = globalThis.crypto.getRandomValues(new Uint8Array(length))
                  return Promise.resolve({ randomValues: values.buffer })
                },
              }
            `,
            loader: 'js',
          }))
        },
      },
    ],
    write: false,
  })
  const encoded = Buffer.from(result.outputFiles[0].text).toString('base64')
  const encryptedModule = await import(`data:text/javascript;base64,${encoded}`)

  assert.equal(encryptedModule.plaintext, '112233')
})
