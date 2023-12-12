package config

import "github.com/sgostarter/libcomponents/account"

type Config struct {
	Debug  bool   `yaml:"debug" json:"debug"`
	Listen string `yaml:"listen" json:"listen"`

	AccountConfig account.Config `yaml:"accountConfig" json:"accountConfig"`
}

func (cfg *Config) Valid() bool {
	return cfg.Listen != ""
}
