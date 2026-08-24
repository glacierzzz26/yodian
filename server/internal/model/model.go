// Package model 对应设计 7.1 全部表 + 补充 employees。
// 迁移由 SQL 管理（golang-migrate 风格），本包仅作 CRUD 模型。
package model

import (
	"time"

	"gorm.io/datatypes"
)

type Shop struct {
	ID        int64  `gorm:"primaryKey"`
	Name      string `gorm:"size:128;not null"`
	Address   *string
	CreatedAt time.Time
}

type TableArea struct {
	ID             int64   `gorm:"primaryKey"`
	ShopID         int64   `gorm:"not null;index"`
	Name           string  `gorm:"size:32;not null"`
	SortOrder      int     `gorm:"not null;default:0"`
	DefaultPayMode *string
	IsActive       bool `gorm:"not null;default:true"`
}

type Table struct {
	ID        int64  `gorm:"primaryKey"`
	ShopID    int64  `gorm:"not null;index"`
	AreaID    *int64 `gorm:"index"`
	TableNo   string `gorm:"size:32;not null"`
	Seats     int    `gorm:"not null;default:2"`
	Status    string `gorm:"size:16;not null;default:empty"` // empty / occupied（派生，3.5）
	PayMode   string `gorm:"size:16;not null;default:prepay"` // prepay/postpay/frontend
	SortOrder int    `gorm:"not null;default:0"`
	Remark    *string
	QrVersion int    `gorm:"not null;default:1"`
	QrcodeURL *string
	CreatedAt time.Time
}

type Customer struct {
	ID              int64      `gorm:"primaryKey"`
	Channel         string     `gorm:"size:16;not null;index:idx_cust_channel_uid,unique"` // wechat/alipay
	ChannelUID      string     `gorm:"size:128;not null;index:idx_cust_channel_uid,unique"`
	Nickname        *string
	Phone           *string
	PhoneVerifiedAt *time.Time
	CreatedAt       time.Time
}

type TableSession struct {
	ID            int64      `gorm:"primaryKey"`
	ShopID        int64      `gorm:"not null;index"`
	TableID       int64      `gorm:"not null;index"`
	PayMode       string     `gorm:"size:16;not null"` // 开台时从 tables 快照（3.7）
	Status        string     `gorm:"size:16;not null;default:active"`
	LockedForBill bool       `gorm:"not null;default:false"`
	Pax           int        `gorm:"not null;default:2"`
	OpenedAt      time.Time
	ClosedAt      *time.Time
}

type Bill struct {
	ID             int64      `gorm:"primaryKey"`
	ShopID         int64      `gorm:"not null;index"`
	SessionID      int64      `gorm:"not null;index"`
	TotalAmount    Money      `gorm:"type:numeric(10,2);not null"`
	DiscountAmount Money      `gorm:"type:numeric(10,2);not null;default:0"`
	PayableAmount  Money      `gorm:"type:numeric(10,2);not null"`
	Status         string     `gorm:"size:16;not null"` // pending/paid/closed
	PayChannel     *string
	OutTradeNo     *string
	OperatorID     *int64
	CreatedAt      time.Time
	PaidAt         *time.Time
}

type PerHeadCharge struct {
	ID         int64  `gorm:"primaryKey"`
	ShopID     int64  `gorm:"not null;index"`
	Name       string `gorm:"size:64;not null"`
	Price      Money  `gorm:"type:numeric(10,2);not null"`
	IsDefault  bool   `gorm:"not null;default:true"`
	IsRequired bool   `gorm:"not null;default:false"`
	IsActive   bool   `gorm:"not null;default:true"`
	Sort       int    `gorm:"not null;default:0"`
	CreatedAt  time.Time
}

type Category struct {
	ID     int64  `gorm:"primaryKey"`
	ShopID int64  `gorm:"not null;index"`
	Name   string `gorm:"size:64;not null"`
	Sort   int    `gorm:"not null;default:0"`
}

type Dish struct {
	ID         int64  `gorm:"primaryKey"`
	ShopID     int64  `gorm:"not null;index"`
	CategoryID int64  `gorm:"not null;index"`
	Name       string `gorm:"size:128;not null"`
	Price      Money  `gorm:"type:numeric(10,2);not null"`
	Specs      datatypes.JSON
	IsSoldOut  bool    `gorm:"not null;default:false"`
	ImageURL   *string `gorm:"size:512"`
	Sort       int     `gorm:"not null;default:0"`
}

type Order struct {
	ID             int64      `gorm:"primaryKey"`
	ShopID         int64      `gorm:"not null;index"`
	TableID        *int64     `gorm:"index"`
	SessionID      *int64     `gorm:"index"`
	CustomerID     *int64     `gorm:"index"`
	BillID         *int64     `gorm:"index"`
	PayMode        string     `gorm:"size:16;not null;default:prepay"`
	OrderType      string     `gorm:"size:16;not null;default:dine_in"`
	PickupNo       *string
	Source         string     `gorm:"size:16;not null;default:online"`
	Status         string     `gorm:"size:32;not null;index"` // 含 partially_refunded(18) 等全枚举
	PayChannel     *string    `gorm:"size:16"`
	OperatorID     *int64
	TotalAmount    Money      `gorm:"type:numeric(10,2);not null"`
	DiscountAmount Money      `gorm:"type:numeric(10,2);not null;default:0"`
	PaidAmount     *Money     `gorm:"type:numeric(10,2)"`
	MemberID       *int64
	OutTradeNo     *string    `gorm:"size:64;uniqueIndex"`
	CreatedAt      time.Time
	PaidAt         *time.Time
	FinishedAt     *time.Time

	Items []OrderItem `gorm:"foreignKey:OrderID"`
}

type OrderItem struct {
	ID         int64  `gorm:"primaryKey"`
	OrderID    int64  `gorm:"not null;index"`
	DishID     *int64
	ItemType   string `gorm:"size:16;not null;default:dish"` // dish / per_head
	Name       string `gorm:"size:128;not null"`
	Price      Money  `gorm:"type:numeric(10,2);not null"`
	Qty        int    `gorm:"not null;default:1"`
	Specs      datatypes.JSON
	Remark     *string
	IsServed   bool `gorm:"not null;default:false"`
	IsRefunded bool `gorm:"not null;default:false"`
}

type Payment struct {
	ID              int64  `gorm:"primaryKey"`
	OrderID         *int64 `gorm:"index"`
	BillID          *int64 `gorm:"index"`
	OutTradeNo      string `gorm:"size:64;not null;uniqueIndex"`
	Channel         string `gorm:"size:16;not null"`
	ChannelTradeNo  *string
	Amount          Money `gorm:"type:numeric(10,2);not null"`
	Status          string `gorm:"size:16;not null"` // pending/success/failed/closed
	CallbackRaw     datatypes.JSON
	CreatedAt       time.Time
}

type Refund struct {
	ID           int64  `gorm:"primaryKey"`
	OrderID      *int64 `gorm:"index"`
	BillID       *int64 `gorm:"index"`
	PaymentID    *int64 `gorm:"index"`
	OutRefundNo  string `gorm:"size:64;not null;uniqueIndex"`
	Channel      string `gorm:"size:16;not null"`
	Amount       Money  `gorm:"type:numeric(10,2);not null"`
	Status       string `gorm:"size:16;not null"`
	Reason       *string
	OperatorID   *int64
	CallbackRaw  datatypes.JSON
	CreatedAt    time.Time
}

type PrintTask struct {
	ID         int64  `gorm:"primaryKey"`
	OrderID    *int64 `gorm:"index"`
	BillID     *int64 `gorm:"index"`
	PrinterSN  string `gorm:"size:64;not null"`
	Station    *string
	Status     string `gorm:"size:16;not null;default:pending"`
	RetryCount int    `gorm:"not null;default:0"`
	CreatedAt  time.Time
}

type ServiceCall struct {
	ID         int64      `gorm:"primaryKey"`
	ShopID     int64      `gorm:"not null;index"`
	SessionID  int64      `gorm:"not null;index"`
	TableID    int64      `gorm:"not null;index"`
	CustomerID *int64
	Reason     *string
	Status     string     `gorm:"size:16;not null;default:pending"`
	OperatorID *int64
	CreatedAt  time.Time
	ResolvedAt *time.Time
}

// AuditLog 操作日志（设计 16.2：敏感操作追责，阶段 2.4）
type AuditLog struct {
	ID         int64          `gorm:"primaryKey" json:"id"`
	ShopID     int64          `gorm:"not null;index" json:"shop_id"`
	OperatorID *int64         `gorm:"index" json:"operator_id"`
	Role       *string        `json:"role"`
	Action     string         `gorm:"size:64;not null" json:"action"`
	TargetType *string        `gorm:"size:32" json:"target_type"`
	TargetID   *int64         `json:"target_id"`
	Detail     datatypes.JSON `gorm:"type:jsonb" json:"detail"`
	IP         *string        `json:"ip"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Employee 补充表（设计 7.1 缺失，按 16.2 员工 JWT + 前端 Staff 契约补齐）
type Employee struct {
	ID           int64      `gorm:"primaryKey"`
	ShopID       int64      `gorm:"not null;index"`
	Name         string     `gorm:"size:32;not null"`
	EmployeeNo   string     `gorm:"size:32;not null;index:idx_emp_no_shop,unique"`
	PasswordHash string     `gorm:"size:128;not null"`
	Role         string     `gorm:"size:16;not null"` // owner / cashier / kitchen
	Phone        *string    `gorm:"size:20"`
	Status       string     `gorm:"size:16;not null;default:active"` // active / disabled
	LastLoginAt  *time.Time
	CreatedAt    time.Time
}
