// Package config 使用 viper 管理应用配置。
// 配置加载优先级：环境变量（MEDICAL_前缀）> config.yaml 文件 > 默认值。
// MVP 阶段无需 config.yaml，所有必要参数通过环境变量注入即可启动。
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 是应用全部配置的聚合根，通过 mapstructure tag 映射到 viper。
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Encryption EncryptionConfig `mapstructure:"encryption"`
	Wechat     WechatConfig     `mapstructure:"wechat"`
	SMS        SMSConfig        `mapstructure:"sms"`
	Log        LogConfig        `mapstructure:"log"`
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Port string `mapstructure:"port"` // 监听端口，默认 8080
	Mode string `mapstructure:"mode"` // 运行模式: debug / release / test
}

// DatabaseConfig MySQL 数据库连接配置。
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	Charset         string `mapstructure:"charset"`           // 默认 utf8mb4
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`    // 最大空闲连接数
	MaxOpenConns    int    `mapstructure:"max_open_conns"`    // 最大活跃连接数
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // 连接最大存活时间（秒）
}

// DSN 拼装 MySQL 连接字符串。
// 格式: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.Charset)
}

// RedisConfig Redis 连接配置。
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"` // Redis 数据库编号，默认 0
}

// Addr 返回 Redis 地址 "host:port"。
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// JWTConfig JWT 鉴权配置。
// Access Token: 7 天短期凭证，用于 API 请求。
// Refresh Token: 30 天长期凭证，仅用于换取新 Token 对。
type JWTConfig struct {
	Secret          string `mapstructure:"secret"`            // HMAC-SHA256 签名密钥（必须配置环境变量）
	AccessTokenTTL  int    `mapstructure:"access_token_ttl"`  // Access Token 有效期（小时），默认 168
	RefreshTokenTTL int    `mapstructure:"refresh_token_ttl"` // Refresh Token 有效期（小时），默认 720
}

// GetAccessTokenTTL 返回 Access Token 有效期（time.Duration）。
func (j *JWTConfig) GetAccessTokenTTL() time.Duration {
	return time.Duration(j.AccessTokenTTL) * time.Hour
}

// GetRefreshTokenTTL 返回 Refresh Token 有效期（time.Duration）。
func (j *JWTConfig) GetRefreshTokenTTL() time.Duration {
	return time.Duration(j.RefreshTokenTTL) * time.Hour
}

// EncryptionConfig 数据加密配置。
type EncryptionConfig struct {
	Key string `mapstructure:"key"` // AES-256 密钥，必须 32 字节
}

// WechatConfig 微信小程序配置。
type WechatConfig struct {
	Mode      string `mapstructure:"mode"`       // 实现模式：mock（开发）/ real（生产）
	AppID     string `mapstructure:"app_id"`     // 小程序 AppID
	AppSecret string `mapstructure:"app_secret"` // 小程序 AppSecret
}

// SMSConfig 腾讯云短信配置（P1 兜底通知通道）。
type SMSConfig struct {
	SecretID   string `mapstructure:"secret_id"`
	SecretKey  string `mapstructure:"secret_key"`
	SDKAppID   string `mapstructure:"sdk_app_id"`
	SignName   string `mapstructure:"sign_name"`   // 短信签名
	TemplateID string `mapstructure:"template_id"` // 短信模板 ID
}

// LogConfig 日志配置。
type LogConfig struct {
	Level      string `mapstructure:"level"`       // debug / info / warn / error
	Filename   string `mapstructure:"filename"`    // 日志文件路径（空 = 仅控制台输出）
	MaxSize    int    `mapstructure:"max_size"`    // 单个日志文件最大大小（MB），默认 10
	MaxBackups int    `mapstructure:"max_backups"` // 最多保留旧日志文件数，默认 5
	MaxAge     int    `mapstructure:"max_age"`     // 旧日志文件最多保留天数，默认 30
}

// Load 从配置文件和环境变量加载全部配置。
// configPath 为空时自动搜索当前目录下的 config.yaml。
// 环境变量前缀为 MEDICAL_，如 MEDICAL_JWT_SECRET 覆盖 jwt.secret。
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// --- 默认值（开发环境开箱即用）---
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.mode", "debug")
	v.SetDefault("database.host", "127.0.0.1")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.charset", "utf8mb4")
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.conn_max_lifetime", 3600)
	v.SetDefault("redis.host", "127.0.0.1")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.db", 0)
	v.SetDefault("jwt.access_token_ttl", 168)  // 7 天
	v.SetDefault("jwt.refresh_token_ttl", 720) // 30 天
	v.SetDefault("log.level", "debug")
	v.SetDefault("log.filename", "logs/app.log")
	v.SetDefault("log.max_size", 10)
	v.SetDefault("log.max_backups", 5)
	v.SetDefault("log.max_age", 30)

	if configPath != "" {
		v.SetConfigFile(configPath)
		v.SetConfigType("yaml")
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
	}

	// 环境变量覆盖（MEDICAL_ 前缀自动映射到配置路径）
	v.SetEnvPrefix("MEDICAL")
	v.AutomaticEnv()

	// 显式绑定关键环境变量
	_ = v.BindEnv("server.port", "MEDICAL_SERVER_PORT")
	_ = v.BindEnv("database.host", "MEDICAL_DB_HOST")
	_ = v.BindEnv("database.port", "MEDICAL_DB_PORT")
	_ = v.BindEnv("database.user", "MEDICAL_DB_USER")
	_ = v.BindEnv("database.password", "MEDICAL_DB_PASSWORD")
	_ = v.BindEnv("database.dbname", "MEDICAL_DB_NAME")
	_ = v.BindEnv("jwt.secret", "MEDICAL_JWT_SECRET")
	_ = v.BindEnv("encryption.key", "MEDICAL_ENCRYPTION_KEY")
	_ = v.BindEnv("wechat.app_id", "MEDICAL_WECHAT_APP_ID")
	_ = v.BindEnv("wechat.app_secret", "MEDICAL_WECHAT_APP_SECRET")
	_ = v.BindEnv("redis.host", "MEDICAL_REDIS_HOST")
	_ = v.BindEnv("redis.password", "MEDICAL_REDIS_PASSWORD")

	// 配置文件不存在时不视为错误，回退到默认值 + 环境变量
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// JWT secret 安全校验：为空时生成随机密钥（仅开发环境）。
	// 生产环境必须通过 MEDICAL_JWT_SECRET 环境变量设置强密钥。
	if cfg.JWT.Secret == "" {
		cfg.JWT.Secret = randomSecret()
	}

	return &cfg, nil
}

// randomSecret 生成 32 字节随机密钥，用于开发环境兜底。
// 注意：每次启动生成不同密钥，会导致已签发 Token 全部失效，仅限本地开发。
func randomSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "dev-insecure-random-secret"
	}
	return hex.EncodeToString(b)
}
