package config

import (
	_ "embed"
	"fmt"

	"github.com/bizshuk/gosdk/config"
	"github.com/spf13/viper"
)

//go:embed default_settings.json
var defaultSettingsJSON string

// GlobalSettings 全域設定實例
var GlobalSettings *Settings

// Settings 定義所有設定項目
type Settings struct {
	CheckInterval string      `mapstructure:"check_interval"`
	Timeout       string      `mapstructure:"timeout"`
	MimirURL      string      `mapstructure:"mimir_url"`
	LogLevel      string      `mapstructure:"log_level"`
	Ports         []PortEntry `mapstructure:"ports"`
}

// PortEntry 定義單一連接埠設定
type PortEntry struct {
	Port int    `mapstructure:"port" json:"port"`
	Name string `mapstructure:"name" json:"name"`
	// Health 是 `port health` 的探測目標：HTTP(S) URL，或字面值 "tcp"
	// 表示以 TCP 連線成功作為健康判準。留空則不納入 health 檢查。
	Health string `mapstructure:"health" json:"health,omitempty"`
	// Insecure 跳過 TLS 憑證驗證，供使用自簽憑證的本機服務 opt-in。
	Insecure bool `mapstructure:"insecure" json:"insecure,omitempty"`
}

// Default初始化全域設定
func Default() error {
	config.Default(
		config.WithAppName("port"),
		config.WithDefaultValue(defaultSettingsJSON),
	)

	GlobalSettings = &Settings{}
	if err := viper.Unmarshal(GlobalSettings); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return nil
}
