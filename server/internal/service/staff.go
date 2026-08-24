// 员工凭据校验（设计 16.2/16.4）。完整权限矩阵（角色→路由）在阶段 2.4 落地，
// 此处只负责「工号+密码 → 员工 + 角色」，为清台/核销/退款提供操作者上下文。
package service

import (
	"context"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/yodian/server/internal/model"
	"github.com/yodian/server/internal/pkg/respond"
)

type StaffService struct {
	db     *gorm.DB
	shopID int64
}

func NewStaffService(db *gorm.DB, shopID int64) *StaffService {
	return &StaffService{db: db, shopID: shopID}
}

// Authenticate 校验工号 + 密码，返回员工与角色；账号停用/凭据错误返回对应业务错误。
func (s *StaffService) Authenticate(ctx context.Context, employeeNo, password string) (*model.Employee, *respond.BizErr) {
	var emp model.Employee
	if err := s.db.First(&emp, "shop_id = ? AND employee_no = ?", s.shopID, employeeNo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, respond.ErrCredInvalid
		}
		slog.Error("query employee failed", "err", err)
		return nil, respond.ErrInternal
	}
	if emp.Status != "active" {
		return nil, respond.ErrStaffDisabled
	}
	if bcrypt.CompareHashAndPassword([]byte(emp.PasswordHash), []byte(password)) != nil {
		return nil, respond.ErrCredInvalid
	}
	// 登录成功：记录最近登录
	if err := s.db.Model(&emp).Update("last_login_at", gorm.Expr("now()")).Error; err != nil {
		slog.Warn("update last_login_at failed", "err", err)
	}
	return &emp, nil
}
