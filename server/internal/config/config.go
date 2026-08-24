// Package config 加载 .env（8 章全部外部依赖变量）与 JWT/DB/Redis 配置
package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Env      string
	HTTPPort string
	ShopID   int64 // 单租户店铺常量（部署即一家店，seed 为 1）

	DatabaseURL string
	RedisAddr   string
	RedisPass   string

	JWTCustomerSecret string
	JWTCustomerTTLH   int // 小时，顾客 token 长期有效（16.2 静默续期）
	JWTStaffSecret    string
	JWTStaffAccessH   int // 小时
	JWTStaffRefreshD  int // 天

	Wechat WechatCfg
	Alipay AlipayCfg
	QR     QRCfg
	SMS    SMSCfg

	EnableWechatPay bool
	EnableAlipay    bool

	MockWechatPay bool
	MockAlipay    bool
	MockPrinter   bool
}

type WechatCfg struct {
	AppID     string
	MchID     string
	APIV3Key  string
	SerialNo  string
	NotifyURL string
}

type AlipayCfg struct {
	AppID     string
	Gateway   string
	SignType  string
	NotifyURL string
	Sandbox   bool
}

type QRCfg struct {
	SignSecret string
	BaseURL    string
	TTLDays    int
}

type SMSCfg struct {
	Provider string
	CodeTTL  int // 秒
	Limit    int
}

// Load 读取 .env（若存在）并解析环境变量
func Load() *Config {
	// .env 可选（生产用真实环境变量注入）；本地开发从 .env 读
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file, using process env")
	}
	return &Config{
		Env:      get("APP_ENV", "dev"),
		HTTPPort: get("HTTP_PORT", "8080"),
		ShopID:   int64(getInt("SHOP_ID", 1)),

		DatabaseURL: get("DATABASE_URL", "postgres://yodian:yodian_dev@localhost:5432/yodian?sslmode=disable"),
		RedisAddr:   get("REDIS_ADDR", "localhost:6379"),
		RedisPass:   get("REDIS_PASSWORD", ""),

		JWTCustomerSecret: get("JWT_CUSTOMER_SECRET", "__change_me_customer__"),
		JWTCustomerTTLH:   getInt("JWT_CUSTOMER_TTL_HOURS", 720),
		JWTStaffSecret:    get("JWT_STAFF_SECRET", "__change_me_staff__"),
		JWTStaffAccessH:   getInt("JWT_STAFF_ACCESS_HOURS", 2),
		JWTStaffRefreshD:  getInt("JWT_STAFF_REFRESH_DAYS", 7),

		Wechat: WechatCfg{
			AppID:     get("WECHAT_APPID", "wx_placeholder"),
			MchID:     get("WECHAT_MCH_ID", ""),
			APIV3Key:  get("WECHAT_APIV3_KEY", ""),
			SerialNo:  get("WECHAT_SERIAL_NO", ""),
			NotifyURL: get("WECHAT_NOTIFY_URL", ""),
		},
		Alipay: AlipayCfg{
			AppID:     get("ALIPAY_APPID", ""),
			Gateway:   get("ALIPAY_GATEWAY", "https://openapi.alipay.com/gateway.do"),
			SignType:  get("ALIPAY_SIGN_TYPE", "RSA2"),
			NotifyURL: get("ALIPAY_NOTIFY_URL", ""),
			Sandbox:   getBool("ALIPAY_SANDBOX", false),
		},
		QR: QRCfg{
			SignSecret: get("QR_SIGN_SECRET", "__change_me__"),
			BaseURL:    get("QR_BASE_URL", "http://localhost:8080/t"),
			TTLDays:    getInt("QR_TTL_DAYS", 365),
		},
		SMS: SMSCfg{
			Provider: get("SMS_PROVIDER", "mock"),
			CodeTTL:  getInt("SMS_CODE_TTL_SECONDS", 300),
			Limit:    getInt("SMS_CODE_LIMIT_PER_PHONE", 3),
		},

		EnableWechatPay: getBool("ENABLE_WECHAT_PAY", true),
		EnableAlipay:    getBool("ENABLE_ALIPAY", true),

		MockWechatPay: getBool("MOCK_WECHAT_PAY", true),
		MockAlipay:    getBool("MOCK_ALIPAY", true),
		MockPrinter:   getBool("MOCK_PRINTER", true),
	}
}

func get(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
