// Package landing 中间页 H5（设计 4.3 兜底页）：单文件 HTML，UA 分流 + 引导 + 二维码无效屏。
// 由 server 通过 go:embed 嵌入二进制，挂根路径 /t（非 /api）。要求极轻（预算 ≤400ms）。
package landing

import "embed"

//go:embed index.html
var FS embed.FS

// IndexHTML 返回中间页 HTML（服务端以 text/html 返回）
func IndexHTML() []byte {
	b, _ := FS.ReadFile("index.html")
	return b
}
