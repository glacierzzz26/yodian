// Package auth JWT 签发/解析（设计 16.2）。
// 顾客 token payload {cid,ch,sid,exp}；员工 token payload {oid,role,sid,typ,exp}。
// openid / user_id 仅服务端持有，不下发。
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomerClaims 顾客 token
type CustomerClaims struct {
	CID int64  `json:"cid"`
	CH  string `json:"ch"` // wechat / alipay
	SID int64  `json:"sid"`
	jwt.RegisteredClaims
}

// StaffClaims 员工 token
type StaffClaims struct {
	OID  int64  `json:"oid"`
	Role string `json:"role"` // owner / cashier / kitchen
	SID  int64  `json:"sid"`
	Typ  string `json:"typ"` // access / refresh
	jwt.RegisteredClaims
}

var ErrInvalidToken = errors.New("invalid token")

// SignCustomer 签发顾客 token
func SignCustomer(secret string, cid, sid int64, ch string, ttl time.Duration) (string, error) {
	claims := CustomerClaims{
		CID: cid, CH: ch, SID: sid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseCustomer 解析并校验顾客 token
func ParseCustomer(secret, token string) (*CustomerClaims, error) {
	var c CustomerClaims
	if _, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})); err != nil {
		return nil, ErrInvalidToken
	}
	return &c, nil
}

// SignStaff 签发员工 token（typ=access/refresh）
func SignStaff(secret string, oid, sid int64, role, typ string, ttl time.Duration) (string, error) {
	claims := StaffClaims{
		OID: oid, Role: role, SID: sid, Typ: typ,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseStaff 解析并校验员工 token
func ParseStaff(secret, token string) (*StaffClaims, error) {
	var c StaffClaims
	if _, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})); err != nil {
		return nil, ErrInvalidToken
	}
	return &c, nil
}
