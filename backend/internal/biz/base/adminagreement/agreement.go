package adminagreement

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/liujitcn/kratos-core/errorsx"
)

// Policy 保存管理端独立的协议正文和版本。
type Policy struct {
	Service        string // 管理端服务协议正文。
	ServiceVersion string // 管理端服务协议版本。
	Privacy        string // 管理端隐私政策正文。
	PrivacyVersion string // 管理端隐私政策版本。
}

// Receipt 保存服务端确认的协议版本、正文摘要和确认时间。
type Receipt struct {
	ServiceVersion string    `json:"service_version"` // 确认的管理端服务协议版本。
	PrivacyVersion string    `json:"privacy_version"` // 确认的管理端隐私政策版本。
	ServiceDigest  string    `json:"service_digest"`  // 管理端服务协议正文的 SHA-256 摘要。
	PrivacyDigest  string    `json:"privacy_digest"`  // 管理端隐私政策正文的 SHA-256 摘要。
	AcceptedAt     time.Time `json:"accepted_at"`     // 服务端记录的主动确认时间。
}

type receiptKey struct{}
type captureKey struct{}

// capture 将已完成认证的确认结果传回外层审计。
type capture struct {
	receipt *Receipt
}

// Confirm 校验主动同意的管理端协议并生成服务端确认记录。
func Confirm(policy Policy, accepted bool, serviceVersion, privacyVersion string, now time.Time) (Receipt, error) {
	if !accepted {
		return Receipt{}, errorsx.InvalidArgument("请先阅读并同意管理端服务协议和隐私政策")
	}
	if policy.Service == "" || policy.Privacy == "" || policy.ServiceVersion == "" || policy.PrivacyVersion == "" {
		return Receipt{}, errorsx.PermissionDenied("管理端服务协议和隐私政策尚未配置")
	}
	if serviceVersion != policy.ServiceVersion || privacyVersion != policy.PrivacyVersion {
		return Receipt{}, errorsx.Conflict("管理端协议已更新，请重新阅读并同意")
	}
	return Receipt{
		ServiceVersion: policy.ServiceVersion,
		PrivacyVersion: policy.PrivacyVersion,
		ServiceDigest:  fmt.Sprintf("%x", sha256.Sum256([]byte(policy.Service))),
		PrivacyDigest:  fmt.Sprintf("%x", sha256.Sum256([]byte(policy.Privacy))),
		AcceptedAt:     now,
	}, nil
}

// Validate 校验 OAuth 或 MFA 继续认证时协议仍与确认时一致。
func Validate(policy Policy, receipt Receipt) error {
	current, err := Confirm(policy, true, receipt.ServiceVersion, receipt.PrivacyVersion, receipt.AcceptedAt)
	if err != nil {
		return err
	}
	if receipt.AcceptedAt.IsZero() || current.ServiceDigest != receipt.ServiceDigest || current.PrivacyDigest != receipt.PrivacyDigest {
		return errorsx.Conflict("管理端协议已更新，请重新阅读并同意")
	}
	return nil
}

// WithCapture 为当前请求创建协议确认审计接收器。
func WithCapture(ctx context.Context) context.Context {
	return context.WithValue(ctx, captureKey{}, &capture{})
}

// WithReceipt 将服务端校验后的确认附加到本次管理端认证流程。
func WithReceipt(ctx context.Context, receipt Receipt) context.Context {
	return context.WithValue(ctx, receiptKey{}, receipt)
}

// ReceiptFromContext 返回当前认证流程携带的协议确认。
func ReceiptFromContext(ctx context.Context) (Receipt, bool) {
	receipt, ok := ctx.Value(receiptKey{}).(Receipt)
	return receipt, ok
}

// Record 在认证完成后向外层审计提交协议确认副本。
func Record(ctx context.Context) {
	receipt, ok := ReceiptFromContext(ctx)
	if !ok {
		return
	}
	if target, exists := ctx.Value(captureKey{}).(*capture); exists {
		target.receipt = &receipt
	}
}

// FromContext 返回本次请求已完成认证的协议确认。
func FromContext(ctx context.Context) *Receipt {
	if target, ok := ctx.Value(captureKey{}).(*capture); ok {
		return target.receipt
	}
	return nil
}
