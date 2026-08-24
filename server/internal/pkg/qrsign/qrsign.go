// Package qrsign 桌贴二维码签名（设计 4.4）。
// QR 原文 = s{shop_id}.t{table_id}.v{qr_version}，HMAC-SHA256(QR_SIGN_SECRET)。
// 签名用于防止伪造桌台码：顾客菜单接口必须携带合法 sig 才放行。
package qrsign

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

// Payload 构造 QR 原文
func Payload(shopID, tableID int64, version int) string {
	return fmt.Sprintf("s%d.t%d.v%d", shopID, tableID, version)
}

// Sign 生成 hex HMAC-SHA256 签名
func Sign(secret string, shopID, tableID int64, version int) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(Payload(shopID, tableID, version)))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify 常量时间比较，防时序侧信道
func Verify(secret string, shopID, tableID int64, version int, sig string) bool {
	got := Sign(secret, shopID, tableID, version)
	if len(got) != len(sig) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(sig)) == 1
}
