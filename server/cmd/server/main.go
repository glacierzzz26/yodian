// 悦点（Yodian）商家端后端服务入口
package main

import (
	"log/slog"
	"os"

	"github.com/yodian/server/internal/config"
	"github.com/yodian/server/internal/pay"
	"github.com/yodian/server/internal/router"
	"github.com/yodian/server/internal/store"
)

func main() {
	cfg := config.Load()

	level := slog.LevelInfo
	if cfg.Env == "dev" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	db, err := store.OpenDB(cfg.DatabaseURL)
	if err != nil {
		slog.Error("open db failed", "err", err)
		os.Exit(1)
	}
	slog.Info("postgres connected")

	rdb, err := store.OpenRedis(cfg.RedisAddr, cfg.RedisPass)
	if err != nil {
		slog.Error("open redis failed", "err", err)
		os.Exit(1)
	}
	slog.Info("redis connected")

	if err := store.MigrateUp(db); err != nil {
		slog.Error("migrate failed", "err", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	// 支付网关注册：mock 实现（开发期默认开）；真实微信/支付宝接入在 2.4 依据 MOCK 开关替换
	gw := pay.GatewayRegistry{}
	if cfg.MockWechatPay {
		gw[pay.ChannelWechat] = pay.NewMockGateway(pay.ChannelWechat)
	}
	if cfg.MockAlipay {
		gw[pay.ChannelAlipay] = pay.NewMockGateway(pay.ChannelAlipay)
	}

	r := router.New(cfg, db, rdb, gw)
	addr := ":" + cfg.HTTPPort
	slog.Info("server listening", "addr", addr, "env", cfg.Env)
	if err := r.Run(addr); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}
