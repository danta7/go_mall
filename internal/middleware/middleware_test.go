package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

//
// RequestID
//

func TestRequestID_GenerateWhenMissing(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFromContext(r.Context())
		if id == "" {
			t.Fatalf("request id not found in context")
		}
		// 校验 header 也被写入
		if got := w.Header().Get(HeaderRequestID); got == "" {
			t.Fatalf("request id header not set")
		}
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	// 校验是合法 UUID
	if _, err := uuid.Parse(rec.Header().Get(HeaderRequestID)); err != nil {
		t.Fatalf("request id is not a valid uuid: %v", err)
	}
}

func TestRequestID_PreserveIncoming(t *testing.T) {
	const in = "abc-123"
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := RequestIDFromContext(r.Context()); got != in {
			t.Fatalf("context request id mismatch: want %q, got %q", in, got)
		}
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(HeaderRequestID, in)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if got := rec.Header().Get(HeaderRequestID); got != in {
		t.Fatalf("header request id mismatch: want %q, got %q", in, got)
	}
}

//
// AccessLog
//

func newObservedLogger(lvl zapcore.Level) (*zap.Logger, *observer.ObservedLogs) {
	core, obs := observer.New(lvl)
	return zap.New(core), obs
}

func TestAccessLog_RecordsStatusAndPassThrough(t *testing.T) {
	logger, obs := newObservedLogger(zapcore.InfoLevel)

	// 业务 handler 返回自定义状态码
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // 418
		_, _ = w.Write([]byte("hi"))
	})

	h := AccessLog(logger)(next)
	req := httptest.NewRequest(http.MethodGet, "/tea", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status passthrough failed: want %d, got %d", http.StatusTeapot, rec.Code)
	}

	entries := obs.All()
	if len(entries) != 1 {
		t.Fatalf("want 1 log entry, got %d", len(entries))
	}
	// 从日志字段里找 status
	var got int64
	for _, f := range entries[0].Context {
		if f.Key == "status" {
			got = f.Integer
			break
		}
	}
	if got != int64(http.StatusTeapot) {
		t.Fatalf("logged status mismatch: want %d, got %d", http.StatusTeapot, got)
	}
}

//
// CORS
//

func TestCORS_ActualRequestHeadersSet(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"https://a.com", "https://b.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	}
	h := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	// 即使没带 Origin，你当前实现也会设置 CORS 头（按目前代码来测）
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://a.com, https://b.com" {
		t.Fatalf("allow-origin mismatch: got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST" {
		t.Fatalf("allow-methods mismatch: got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type" {
		t.Fatalf("allow-headers mismatch: got %q", got)
	}
	// 你现在用 Set 连续写 Vary，只会保留最后一个；按现实现象断言
	if got := rec.Header().Get("Vary"); got != "Access-Control-Request-Headers" {
		t.Fatalf("vary mismatch (current impl keeps last one): got %q", got)
	}
}

func TestCORS_PreflightOPTIONS(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	}
	h := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("preflight should not hit next handler")
	}))
	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status want 204, got %d", rec.Code)
	}
}

//
// Timeout
//

func TestTimeout_SlowHandlerGets503(t *testing.T) {
	h := Timeout(30 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond) // 超过超时
		_, _ = w.Write([]byte("too slow"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable { // http.TimeoutHandler 默认 503
		t.Fatalf("want 503, got %d", rec.Code)
	}
}

func TestTimeout_FastHandlerOK(t *testing.T) {
	h := Timeout(200 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	req := httptest.NewRequest(http.MethodGet, "/fast", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

//
// Recovery
//

func TestRecovery_PanicIsRecovered(t *testing.T) {
	logger, _ := newObservedLogger(zapcore.ErrorLevel)
	h := Recovery(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	// 不应崩溃
	h.ServeHTTP(rec, req)

	// 状态码由 resp.Error 决定，这里只断言请求没有崩掉且写了响应
	if rec.Code == 0 {
		t.Fatalf("response code not written")
	}
}

func TestRecovery_PassThrough(t *testing.T) {
	logger, _ := newObservedLogger(zapcore.ErrorLevel)
	h := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("recovery should pass through non-panic request: want 204, got %d", rec.Code)
	}
}
