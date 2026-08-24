package model

import (
	"database/sql/driver"
	"fmt"
	"math"
	"strconv"
)

// Money 金额，单位「分」，JSON 出入一律整数（设计 3.2 硬约束，禁止 float 参与业务计算）。
// 入库转 NUMERIC(10,2)「元」，读库转回分。float64 仅作为入库/出库的中间表示，
// 分值除以 100 落在 float64 精确表示范围内（±2^53），无精度损失。
type Money int64

func (m Money) GormDataType() string { return "numeric(10,2)" }

// Value 入库：分 → 元（float64）
func (m Money) Value() (driver.Value, error) {
	return float64(m) / 100, nil
}

// Scan 出库：元 → 分
func (m *Money) Scan(v any) error {
	switch x := v.(type) {
	case nil:
		*m = 0
		return nil
	case float64:
		*m = Money(math.Round(x * 100))
		return nil
	case float32:
		*m = Money(math.Round(float64(x) * 100))
		return nil
	case int64:
		*m = Money(x)
		return nil
	case []byte:
		f, err := strconv.ParseFloat(string(x), 64)
		if err != nil {
			return fmt.Errorf("Money.Scan: %w", err)
		}
		*m = Money(math.Round(f * 100))
		return nil
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return fmt.Errorf("Money.Scan: %w", err)
		}
		*m = Money(math.Round(f * 100))
		return nil
	default:
		return fmt.Errorf("Money.Scan: unsupported type %T", v)
	}
}

// Yuan 以「元」构造 Money（如 NewMoney(12.50) → 1250）
func Yuan(v float64) Money {
	return Money(math.Round(v * 100))
}

// Fen 以「分」构造 Money
func Fen(v int64) Money { return Money(v) }

// ToYuanFloat 仅用于展示/导出，禁止参与计算
func (m Money) ToYuanFloat() float64 { return float64(m) / 100 }
