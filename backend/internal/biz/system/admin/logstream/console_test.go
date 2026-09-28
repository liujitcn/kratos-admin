package logstream

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestStopConsoleCaptureRestoresAndDrains(t *testing.T) {
	originalStdout, err := unix.Dup(int(os.Stdout.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	originalStderr, err := unix.Dup(int(os.Stderr.Fd()))
	if err != nil {
		_ = unix.Close(originalStdout)
		t.Fatal(err)
	}
	defer func() {
		_ = StopConsoleCapture()
		_ = unix.Dup2(originalStdout, int(os.Stdout.Fd()))
		_ = unix.Dup2(originalStderr, int(os.Stderr.Fd()))
		_ = unix.Close(originalStdout)
		_ = unix.Close(originalStderr)
	}()

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdoutReader.Close()
	if err = unix.Dup2(int(stdoutWriter.Fd()), int(os.Stdout.Fd())); err != nil {
		_ = stdoutWriter.Close()
		t.Fatal(err)
	}
	if err = stdoutWriter.Close(); err != nil {
		t.Fatal(err)
	}

	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stderrReader.Close()
	if err = unix.Dup2(int(stderrWriter.Fd()), int(os.Stderr.Fd())); err != nil {
		_ = stderrWriter.Close()
		t.Fatal(err)
	}
	if err = stderrWriter.Close(); err != nil {
		t.Fatal(err)
	}

	hub := &Hub{}
	err = StartConsoleCapture(hub)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fmt.Fprint(os.Stdout, "startup failure on stdout"); err != nil {
		t.Fatal(err)
	}
	if _, err = fmt.Fprint(os.Stderr, "startup failure on stderr"); err != nil {
		t.Fatal(err)
	}
	if err = StopConsoleCapture(); err != nil {
		t.Fatal(err)
	}
	if err = unix.Dup2(originalStdout, int(os.Stdout.Fd())); err != nil {
		t.Fatal(err)
	}
	if err = unix.Dup2(originalStderr, int(os.Stderr.Fd())); err != nil {
		t.Fatal(err)
	}

	stdoutBytes, err := io.ReadAll(stdoutReader)
	if err != nil {
		t.Fatal(err)
	}
	stderrBytes, err := io.ReadAll(stderrReader)
	if err != nil {
		t.Fatal(err)
	}
	if string(stdoutBytes) != "startup failure on stdout" {
		t.Fatalf("stdout capture = %q", stdoutBytes)
	}
	if string(stderrBytes) != "startup failure on stderr" {
		t.Fatalf("stderr capture = %q", stderrBytes)
	}

	hub.mu.RLock()
	defer hub.mu.RUnlock()
	if len(hub.entries) != 2 {
		t.Fatalf("captured entries = %d, want 2", len(hub.entries))
	}
	lines := map[string]string{}
	for _, entry := range hub.entries {
		lines[entry.GetSource()] = strings.TrimSpace(entry.GetLine())
	}
	if lines["stdout"] != "startup failure on stdout" || lines["stderr"] != "startup failure on stderr" {
		t.Fatalf("captured runtime log lines = %#v", lines)
	}
}
