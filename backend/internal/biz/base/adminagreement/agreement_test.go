package adminagreement

import (
	"context"
	"testing"
	"time"
)

// TestConfirmRequiresCurrentAdminAgreements 验证主动确认与管理端协议版本缺一不可。
func TestConfirmRequiresCurrentAdminAgreements(t *testing.T) {
	policy := Policy{Service: "管理端服务正文", ServiceVersion: "service-v1", Privacy: "管理端隐私正文", PrivacyVersion: "privacy-v1"}
	tests := []struct {
		name           string
		policy         Policy
		accepted       bool
		serviceVersion string
		privacyVersion string
	}{
		{name: "未主动同意", policy: policy, serviceVersion: "service-v1", privacyVersion: "privacy-v1"},
		{name: "服务协议版本过期", policy: policy, accepted: true, serviceVersion: "service-v0", privacyVersion: "privacy-v1"},
		{name: "隐私政策版本过期", policy: policy, accepted: true, serviceVersion: "service-v1", privacyVersion: "privacy-v0"},
		{name: "未配置管理端协议", accepted: true, serviceVersion: "service-v1", privacyVersion: "privacy-v1"},
		{name: "正文未配置", policy: Policy{ServiceVersion: "service-v1", Privacy: "管理端隐私正文", PrivacyVersion: "privacy-v1"}, accepted: true, serviceVersion: "service-v1", privacyVersion: "privacy-v1"},
	}
	for _, item := range tests {
		t.Run(item.name, func(t *testing.T) {
			_, err := Confirm(item.policy, item.accepted, item.serviceVersion, item.privacyVersion, time.Now())
			if err == nil {
				t.Fatal("缺少有效确认时仍允许管理端登录")
			}
		})
	}
}

// TestReceiptRejectsChangedAgreements 验证版本或正文更新后不能继续使用旧确认。
func TestReceiptRejectsChangedAgreements(t *testing.T) {
	policy := Policy{Service: "service", ServiceVersion: "service-v1", Privacy: "privacy", PrivacyVersion: "privacy-v1"}
	now := time.Date(2026, 11, 6, 2, 0, 0, 0, time.UTC)
	receipt, err := Confirm(policy, true, "service-v1", "privacy-v1", now)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ServiceVersion != "service-v1" || receipt.PrivacyVersion != "privacy-v1" || !receipt.AcceptedAt.Equal(now) {
		t.Fatalf("确认未记录服务端版本和时间: %+v", receipt)
	}
	if receipt.ServiceDigest != "9df6b026a8c6c26e3c3acd2370a16e93fffdc0015ff5bd879218788025db0280" {
		t.Fatalf("服务正文摘要错误: %s", receipt.ServiceDigest)
	}
	if err = Validate(policy, receipt); err != nil {
		t.Fatalf("当前协议确认被拒绝: %v", err)
	}
	policy.ServiceVersion = "service-v2"
	if err = Validate(policy, receipt); err == nil {
		t.Fatal("服务协议版本更新后旧确认仍然有效")
	}
	policy.ServiceVersion = "service-v1"
	policy.Privacy = "updated privacy"
	if err = Validate(policy, receipt); err == nil {
		t.Fatal("隐私正文更新但版本未变时旧确认仍然有效")
	}
}

// TestCaptureRecordsOnlyCompletedConfirmation 验证外层审计只能读取已完成认证的确认副本。
func TestCaptureRecordsOnlyCompletedConfirmation(t *testing.T) {
	ctx := WithCapture(context.Background())
	receipt := Receipt{ServiceVersion: "service-v1", PrivacyVersion: "privacy-v1", AcceptedAt: time.Now()}
	loginCtx := WithReceipt(ctx, receipt)
	if FromContext(ctx) != nil {
		t.Fatal("尚未完成认证便产生协议确认审计")
	}
	Record(loginCtx)
	receipt.ServiceVersion = "mutated"
	captured := FromContext(ctx)
	if captured == nil || captured.ServiceVersion != "service-v1" {
		t.Fatalf("未保留认证完成时的确认副本: %+v", captured)
	}
}
