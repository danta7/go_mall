package middleware

import "context"

// contextKey 用于在上下文中存取特定键，避免与外部键冲突。
type contextKey string

// 约定的上下文键集合。
const (
	contextKeyRequestID contextKey = "request_id" // 上下文中的 key
)

// withRequestID 将请求 ID 写入上下文。
func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKeyRequestID, id)
}

// RequestIDFromContext 从上下文中读取请求 ID（可能为空）
func RequestIDFromContext(ctx context.Context) string {
	if v := ctx.Value(contextKeyRequestID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
