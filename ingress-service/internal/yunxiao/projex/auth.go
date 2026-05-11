package projex

import "net/http"

// VerifySignature 校验 Projex 自动化 Webhook 请求头中的 X-Projex-Signature。
func VerifySignature(request *http.Request, expected string) bool {
	if expected == "" {
		return true
	}
	return request.Header.Get("X-Projex-Signature") == expected
}
