package flow

import "net/http"

// VerifyToken 校验 Flow Webhook 通知的 Bearer Token。
// Flow 插件可配置自定义 Header；第一版约定使用 Authorization: Bearer <token>。
func VerifyToken(request *http.Request, expected string) bool {
	if expected == "" {
		return true
	}
	return request.Header.Get("Authorization") == "Bearer "+expected
}
