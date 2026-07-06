package config

import (
	"os"

	"github.com/simplefrp/simplefrp/internal/sysutil"
	"github.com/spf13/viper"
)

type Tunnel struct {
	ID         int    `mapstructure:"id" json:"id"`
	LocalPort  int    `mapstructure:"local_port" json:"local_port"`
	PublicPort int    `mapstructure:"public_port" json:"public_port"`
	Status     string `mapstructure:"status" json:"status"`
}

func load(path string, out any) error {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return err
	}
	return v.Unmarshal(out)
}

func writeSecure(path string, content []byte) error {
	if err := os.MkdirAll(sysutil.ConfigDir(), 0750); err != nil {
		return err
	}
	if err := os.WriteFile(path, content, 0640); err != nil {
		return err
	}
	_ = sysutil.ChownToServiceUser(path)
	return nil
}
