package logger

import (
	"os"

	"github.com/medical-monitor/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// New 根据配置创建 zap 日志实例。
// 调试模式：控制台彩色输出。
// 生产模式：控制台 + 文件双写（JSON 格式 + 自动切割）。
func New(cfg *config.Config) *zap.Logger {
	// 1. 日志级别：字符串 → zapcore.Level
	var level zapcore.Level
	switch cfg.Log.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// 2. 编码器配置：ISO8601 时间 + 大写级别 + 短路径调用者
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// 调试模式用彩色控制台格式，生产模式用 JSON
	var encoder zapcore.Encoder
	if cfg.Server.Mode == "debug" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 3. 输出目标：控制台始终输出；配置了文件则同时写文件
	writers := []zapcore.Core{
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level),
	}

	if cfg.Log.Filename != "" {
		// lumberjack 自动切割：按大小 / 按天数 / 保留备份数
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.Log.Filename,   // 日志文件路径
			MaxSize:    cfg.Log.MaxSize,    // 单文件最大大小（MB）
			MaxBackups: cfg.Log.MaxBackups, // 最多保留旧文件数
			MaxAge:     cfg.Log.MaxAge,     // 旧文件最多保留天数
		}
		writers = append(writers,
			zapcore.NewCore(encoder, zapcore.AddSync(fileWriter), level))
	}

	// 合并多个输出（控制台 + 文件）
	core := zapcore.NewTee(writers...)

	// 4. 创建日志实例（附带调用者文件名+行号）
	l := zap.New(core, zap.AddCaller())
	return l
}
