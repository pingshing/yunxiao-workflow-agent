package codeup

import "net/http"

// VerifyToken 校验 Codeup Webhook 请求头中的 X-Codeup-Token。
func VerifyToken(request *http.Request, expected string) bool {
	if expected == "" {
		return true
	}
	return request.Header.Get("X-Codeup-Token") == expected
}
