package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateFromFile_MultiMethodWithContextAndVariadic(t *testing.T) {
	t.Parallel()

	const src = `package sample

import "context"

type Service interface {
	Fetch(ctx context.Context, id string) (value string, err error)
	List(ctx context.Context, tags ...string) []string
	Transform(string, ...int) (string, error)
}
`

	input := writeTempFile(t, "service.go", src)
	out, err := GenerateFromFile(input, "Service")
	if err != nil {
		t.Fatalf("GenerateFromFile error: %v", err)
	}

	generated := string(out)
	requireContains(t, generated, "type ServiceShim struct")
	requireContains(t, generated, "type ServiceMiddleware interface")
	requireContains(t, generated, "type BaseServiceMiddleware struct{}")
	requireContains(t, generated, "type FetchCall struct")
	requireContains(t, generated, "type ListCall struct")
	requireContains(t, generated, "func NewServiceInterceptor(fn func(shimmy.Call, func())) ServiceMiddleware")
	requireContains(t, generated, "for idx := len(s.Middleware) - 1; idx >= 0; idx-- {")

	// Variadic support.
	requireContains(t, generated, "AroundList(ctx context.Context, tags []string, next func(ctx context.Context, tags ...string) []string) []string")
	requireContains(t, generated, "Tags")
	requireContains(t, generated, "[]string")
	requireContains(t, generated, "next(call.Ctx, call.Tags...)")

	// Unnamed params/results are normalized into generated names.
	requireContains(t, generated, "AroundTransform(arg1 string, arg2 []int, next func(arg1 string, arg2 ...int) (string, error)) (string, error)")
	requireContains(t, generated, "Arg1")
	requireContains(t, generated, "Arg2")
	requireContains(t, generated, "[]int")
	requireContains(t, generated, "Result1")
	requireContains(t, generated, "Result2")
}

func TestParseInterfaceFile_UnknownInterface(t *testing.T) {
	t.Parallel()

	const src = `package sample

type Service interface {
	Ping() error
}
`

	input := writeTempFile(t, "service.go", src)
	_, err := ParseInterfaceFile(input, "Missing")
	if err == nil {
		t.Fatal("expected error for unknown interface")
	}
	if !strings.Contains(err.Error(), `unknown interface "Missing"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseInterfaceFile_EmbeddedInterfaceUnsupported(t *testing.T) {
	t.Parallel()

	const src = `package sample

type Base interface { Ping() }

type Service interface {
	Base
}
`

	input := writeTempFile(t, "service.go", src)
	_, err := ParseInterfaceFile(input, "Service")
	if err == nil {
		t.Fatal("expected embedded interface error")
	}
	if !strings.Contains(err.Error(), "embedded interfaces are unsupported in v0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func requireContains(t *testing.T, s, want string) {
	t.Helper()
	if !strings.Contains(s, want) {
		t.Fatalf("generated output missing %q", want)
	}
}
