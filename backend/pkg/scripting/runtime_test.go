package scripting

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRuntimeExecute(t *testing.T) {
	runtime := NewRuntime()
	result, err := runtime.Execute(context.Background(), `ctx.answer = ctx.answer + 1; ctx.message = "ok"; ctx;`, map[string]any{"answer": 1})
	if err != nil {
		t.Fatalf("execute script: %v", err)
	}
	output, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("unexpected output type %T", result)
	}
	answer, err := strconv.ParseFloat(fmt.Sprint(output["answer"]), 64)
	if err != nil || math.Abs(answer-2) > 0.001 {
		t.Fatalf("unexpected answer: %#v", output["answer"])
	}
	if output["message"] != "ok" {
		t.Fatalf("unexpected message: %#v", output["message"])
	}
}

func TestRuntimeTimeout(t *testing.T) {
	runtime := &Runtime{timeout: 20 * time.Millisecond}
	_, err := runtime.Execute(context.Background(), `for (;;) {}`, nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !containsQuotaError(err) {
		t.Fatalf("unexpected timeout error: %v", err)
	}
}

func TestParseContext(t *testing.T) {
	value, err := ParseContext(`{"name":"demo"}`)
	if err != nil {
		t.Fatalf("parse context: %v", err)
	}
	if value["name"] != "demo" {
		t.Fatalf("unexpected context: %#v", value)
	}
	if _, err = ParseContext(`[]`); err == nil {
		t.Fatal("expected object validation error")
	}
}

func containsQuotaError(err error) bool {
	return err != nil && (errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "execution quota exceeded"))
}
