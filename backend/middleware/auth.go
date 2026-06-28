package middleware

import (
	"net/http"
	"strings"
)

// TokenAuth 创建一个中间件，用于校验有效的管理员令牌。
func TokenAuth(validToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 跳过 OPTIONS 请求的认证（CORS 预检）
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// 跳过登录接口的认证
			if r.URL.Path == "/api/login" {
				next.ServeHTTP(w, r)
				return
			}

			// 检查 Authorization 请求头
			authHeader := r.Header.Get("Authorization")
			token := ""

			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				// 同时检查查询参数，以支持 SSE/WebSocket
				token = r.URL.Query().Get("token")
			}

			if token != validToken || validToken == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "Unauthorized - Invalid or missing token"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORS 创建一个添加 CORS 响应头的中间件。
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
