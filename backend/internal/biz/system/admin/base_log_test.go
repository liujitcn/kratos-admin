package biz

import "testing"

func TestLogRecordIDRoundTrip(t *testing.T) {
	const want = "1662579436332053420"

	id, err := parseLogRecordID(want)
	if err != nil {
		t.Fatal(err)
	}
	if got := formatLogRecordID(id); got != want {
		t.Fatalf("日志ID往返后发生变化: got %s, want %s", got, want)
	}
}
