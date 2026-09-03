package oauthsecret

import (
	"bytes"
	"context"
	"testing"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/sdk"
)

type testKey struct{}

func (testKey) Derive(context.Context, string) ([]byte, error) {
	return bytes.Repeat([]byte{1}, 32), nil
}

func TestProtectorRoundTripAndUnprotectedValue(t *testing.T) {
	previousKey := sdk.Runtime.GetKey()
	sdk.Runtime.SetKey(testKey{})
	t.Cleanup(func() {
		sdk.Runtime.SetKey(previousKey)
	})
	protector, err := NewProtector(&configv1.Bootstrap{Authn: &configv1.Authentication{Jwt: &configv1.Authentication_Jwt{Secret: "test-secret"}}})
	if err != nil {
		t.Fatal(err)
	}
	var protected string
	protected, err = protector.Protect("client-secret")
	if err != nil {
		t.Fatal(err)
	}
	if protected == "client-secret" {
		t.Fatal("expected encrypted credential")
	}
	plain, err := protector.Unprotect(protected)
	if err != nil || plain != "client-secret" {
		t.Fatalf("unexpected decrypted credential %q: %v", plain, err)
	}
	if _, err = protector.Unprotect("plain-secret"); err == nil {
		t.Fatal("expected unprotected credential to be rejected")
	}
}
