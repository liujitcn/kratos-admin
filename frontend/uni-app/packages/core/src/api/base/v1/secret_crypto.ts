import { http } from '../../../utils/http'
import type {
  GetSecretPublicKeyRequest,
  GetSecretPublicKeyResponse,
  RevealSecretFieldRequest,
  RevealSecretFieldResponse,
  SecretCryptoService,
} from '../../../rpc/base/v1/secret_crypto'

const SECRET_PUBLIC_KEY_URL = '/v1/base/secret-public-key'
const SECRET_FIELD_REVEAL_URL = '/v1/base/secret-field/reveal'

/** 敏感字段加密公共服务 */
export class SecretCryptoServiceImpl implements SecretCryptoService {
  /** 获取敏感字段临时公钥 */
  GetSecretPublicKey(request: GetSecretPublicKeyRequest): Promise<GetSecretPublicKeyResponse> {
    return http<GetSecretPublicKeyResponse>({
      url: `${SECRET_PUBLIC_KEY_URL}`,
      method: 'GET',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }

  /** 查看密钥字段明文 */
  RevealSecretField(request: RevealSecretFieldRequest): Promise<RevealSecretFieldResponse> {
    return http<RevealSecretFieldResponse>({
      url: `${SECRET_FIELD_REVEAL_URL}`,
      method: 'POST',
      data: request,
    })
  }
}

export const defSecretCryptoService = new SecretCryptoServiceImpl()
