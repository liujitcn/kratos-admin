package sse

import "testing"

func TestNotificationResolveTenantIsolatesTenantAndUser(t *testing.T) {
	stream := NewNotification()
	resolved, err := stream.ResolveTenant("", 42, 7)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != "base.notification:7:42" {
		t.Fatalf("unexpected notification stream %q", resolved)
	}
}

func TestNotificationResolvePrefixedChannelKeepsTenantIsolation(t *testing.T) {
	stream := NewNotification()
	resolved, err := stream.Resolve("7:legacy", 42)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != "base.notification:7:42" {
		t.Fatalf("unexpected notification stream %q", resolved)
	}
}
