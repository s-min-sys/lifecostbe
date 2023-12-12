package main

import (
	"context"
	"fmt"

	"github.com/s-min-sys/lifecostbe/internal/config"
	"github.com/s-min-sys/lifecostbe/internal/server"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libconfig"
	"github.com/sgostarter/liblogrus"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func main() {
	logger := l.NewWrapper(liblogrus.NewLogrusEx(logrus.New()))
	logger.GetLogger().SetLevel(l.LevelDebug)

	var cfg config.Config
	_, _ = libconfig.Load("config.yaml", &cfg)

	cfg.AccountConfig.TokenSignKey = "x"
	cfg.AccountConfig.PasswordHashIterCount = 100

	d, _ := yaml.Marshal(cfg)
	fmt.Println(d)

	server.NewServer(context.Background(), nil, &cfg, logger).Wait()
}
