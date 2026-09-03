package admin

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const baseLogFallbackTestKey = "base-log-fallback-integrity-key"

// TestReplayBaseLogFallbackFilePersistsOffset 验证日志入库回退成功后持久化偏移量并避免重复处理。
func TestReplayBaseLogFallbackFilePersistsOffset(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "admin-log.jsonl")
	key := baseLogFallbackTestKey
	appendBaseLogFallbackTestRecord(t, path, "event-1")
	appendBaseLogFallbackTestRecord(t, path, "event-2")
	replayedIDs := make([]string, 0, 3)
	replay := func(_ context.Context, eventID string, _ []byte) error {
		replayedIDs = append(replayedIDs, eventID)
		return nil
	}

	count, err := replayBaseLogFallbackFile(context.Background(), path, key, replay)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || len(replayedIDs) != 2 {
		t.Fatalf("expected two replayed events, got count=%d ids=%d", count, len(replayedIDs))
	}
	count, err = replayBaseLogFallbackFile(context.Background(), path, key, replay)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected no duplicate replay, got %d", count)
	}
	appendBaseLogFallbackTestRecord(t, path, "event-3")
	count, err = replayBaseLogFallbackFile(context.Background(), path, key, replay)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(replayedIDs) != 3 {
		t.Fatalf("expected one new replayed event, got count=%d ids=%d", count, len(replayedIDs))
	}
}

// TestReplayBaseLogFallbackFileKeepsFailedRecord 验证日志重新写入失败时不推进偏移量，后续执行可以重试。
func TestReplayBaseLogFallbackFileKeepsFailedRecord(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "admin-log.jsonl")
	key := baseLogFallbackTestKey
	appendBaseLogFallbackTestRecord(t, path, "event-1")
	callCount := 0
	replay := func(_ context.Context, _ string, _ []byte) error {
		callCount++
		if callCount == 1 {
			return errors.New("database unavailable")
		}
		return nil
	}

	count, err := replayBaseLogFallbackFile(context.Background(), path, key, replay)
	if err == nil || count != 0 {
		t.Fatalf("expected failed replay, got count=%d err=%v", count, err)
	}
	count, err = replayBaseLogFallbackFile(context.Background(), path, key, replay)
	if err != nil || count != 1 {
		t.Fatalf("expected retry success, got count=%d err=%v", count, err)
	}
}

// TestDecodeBaseLogFallbackRecordRejectsTampering 验证日志入库回退记录被篡改时不会进入重新写入流程。
func TestDecodeBaseLogFallbackRecordRejectsTampering(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "admin-log.jsonl")
	key := baseLogFallbackTestKey
	appendBaseLogFallbackTestRecord(t, path, "event-1")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-2] = 'x'
	if err = os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = decodeBaseLogFallbackRecord(data[:len(data)-1], key); err == nil {
		t.Fatal("expected HMAC verification failure")
	}
}

func appendBaseLogFallbackTestRecord(t *testing.T, path, eventID string) {
	t.Helper()
	eventPayload, err := json.Marshal(baseLogFallbackAdminEvent{
		EventID: eventID,
		Kind:    "login",
		Payload: json.RawMessage(`{"user_name":"admin"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	content, err := json.Marshal(baseLogFallbackContent{
		RecordedAt: "2026-09-02T00:00:00Z",
		Stage:      "queue_unavailable",
		Kind:       "login",
		Operation:  "/base.v1.LoginService/Login",
		Payload:    eventPayload,
	})
	if err != nil {
		t.Fatal(err)
	}
	macValue := hmac.New(sha256.New, []byte(baseLogFallbackTestKey))
	_, _ = macValue.Write(content)
	record, err := json.Marshal(baseLogFallbackRecord{Content: content, HMAC: hex.EncodeToString(macValue.Sum(nil))})
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.Write(append(record, '\n')); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
}
