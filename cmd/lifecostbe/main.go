package main

import (
	"context"
	"flag"

	"github.com/s-min-sys/lifecostbe/internal/config"
	"github.com/s-min-sys/lifecostbe/internal/server"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libconfig"
	"github.com/sgostarter/liblogrus"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func main() {
	var reBuild bool

	flag.BoolVar(&reBuild, "re-build", false, "rebuild statistics")
	flag.Parse()

	logger := l.NewWrapper(liblogrus.NewLogrusEx(logrus.New()))
	logger.GetLogger().SetLevel(l.LevelDebug)

	if reBuild {
		err := server.RebuildBills()
		if err != nil {
			logger.WithFields(l.ErrorField(err)).Fatal("rebuild bills failed")
		} else {
			logger.Info("rebuild bills success")
		}

		return
	}

	var cfg config.Config
	_, _ = libconfig.Load("config.yaml", &cfg)

	cfg.AccountConfig.TokenSignKey = "x"
	cfg.AccountConfig.PasswordHashIterCount = 100

	d, _ := yaml.Marshal(cfg)
	logger.Debug(string(d))

	server.NewServer(context.Background(), nil, &cfg, logger).Wait()
}
