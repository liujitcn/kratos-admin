import service from "@/utils/request";
import type {
  GetSecretPublicKeyRequest,
  GetSecretPublicKeyResponse,
  RevealSecretFieldRequest,
  RevealSecretFieldResponse,
  SecretCryptoService
} from "@/rpc/base/v1/secret_crypto";

const SECRET_PUBLIC_KEY_URL = "/v1/base/secret-public-key";
const SECRET_FIELD_REVEAL_URL = "/v1/base/secret-field/reveal";

/** 敏感字段加密公共服务 */
export class SecretCryptoServiceImpl implements SecretCryptoService {
  /** 获取敏感字段临时公钥 */
  GetSecretPublicKey(request: GetSecretPublicKeyRequest): Promise<GetSecretPublicKeyResponse> {
    return service<GetSecretPublicKeyRequest, GetSecretPublicKeyResponse>({
      url: `${SECRET_PUBLIC_KEY_URL}`,
      method: "get",
      params: request,
      headers: { Authorization: "no-auth" }
    });
  }

  /** 查看密钥字段明文 */
  RevealSecretField(request: RevealSecretFieldRequest): Promise<RevealSecretFieldResponse> {
    return service<RevealSecretFieldRequest, RevealSecretFieldResponse>({
      url: `${SECRET_FIELD_REVEAL_URL}`,
      method: "post",
      data: request
    });
  }
}

export const defSecretCryptoService = new SecretCryptoServiceImpl();
