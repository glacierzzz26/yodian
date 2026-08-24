// 操作日志（设计 16.2：敏感操作记日志，追责与交接班 10.1）。
package middleware

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yodian/server/internal/model"
)

// RecordAudit 记录一条操作日志。detail 为可 JSON 化的补充信息。
func RecordAudit(c *gin.Context, db *gorm.DB, shopID int64, action, targetType string, targetID int64, detail any) {
	var oid *int64
	if v := c.GetInt64("oid"); v > 0 {
		oid = &v
	}
	role := c.GetString("role")
	ip := c.ClientIP()
	raw, _ := json.Marshal(detail)
	rec := model.AuditLog{
		ShopID: shopID, OperatorID: oid,
		Role:       &role,
		Action:     action,
		TargetType: &targetType,
		TargetID:   &targetID,
		Detail:     raw,
		IP:         &ip,
	}
	if err := db.Create(&rec).Error; err != nil {
		// 日志失败不阻断业务
		return
	}
}
