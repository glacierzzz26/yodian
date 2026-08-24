// Package respond 统一响应包络 {code,msg,data}（设计 16.1）
package respond

import "github.com/gin-gonic/gin"

// Resp 统一响应信封：code=0 成功，非 0 业务错误；HTTP 200 含业务错误，
// 400 参数错 / 401 未鉴权 / 403 无权限 / 429 限流 / 5xx 系统
type Resp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

// OK 成功响应
func OK(c *gin.Context, data any) {
	c.JSON(200, Resp{Code: 0, Msg: "ok", Data: data})
}

// OKMsg 成功响应带自定义消息
func OKMsg(c *gin.Context, msg string, data any) {
	c.JSON(200, Resp{Code: 0, Msg: msg, Data: data})
}

// Err 业务错误响应（HTTP 状态取自 BizErr.HTTP）
func Err(c *gin.Context, e *BizErr) {
	c.JSON(e.HTTP, Resp{Code: e.Code, Msg: e.Msg})
}

// ErrStatus 业务错误 + 自定义 HTTP 状态
func ErrStatus(c *gin.Context, http int, code int, msg string) {
	c.JSON(http, Resp{Code: code, Msg: msg})
}

// NotFound 统一 404 信封
func NotFound(c *gin.Context) {
	Err(c, ErrNotFound)
}
