package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	FCM      FCMConfig
}

type AppConfig struct {
	Env  string
	Port string
	Name string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	DSN      string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  int // menit
	RefreshTokenTTL int // hari
}

type FCMConfig struct {
	ServerKey string
}

func Load() *Config {
	v := viper.New()

	v.SetConfigFile(".env")
	v.SetConfigType("env")
	_ = v.ReadInConfig()

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetDefault("APP_ENV", "development")
	v.SetDefault("APP_PORT", "8080")
	v.SetDefault("APP_NAME", "doomslock")
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", "5432")
	v.SetDefault("DB_SSLMODE", "disable")
	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", "6379")
	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("JWT_ACCESS_TOKEN_TTL", 60)
	v.SetDefault("JWT_REFRESH_TOKEN_TTL", 30)

	dbCfg := DatabaseConfig{
		Host:     v.GetString("DB_HOST"),
		Port:     v.GetString("DB_PORT"),
		User:     v.GetString("DB_USER"),
		Password: v.GetString("DB_PASSWORD"),
		Name:     v.GetString("DB_NAME"),
		SSLMode:  v.GetString("DB_SSLMODE"),
	}
	dbCfg.DSN = "host=" + dbCfg.Host +
		" port=" + dbCfg.Port +
		" user=" + dbCfg.User +
		" password=" + dbCfg.Password +
		" dbname=" + dbCfg.Name +
		" sslmode=" + dbCfg.SSLMode

	return &Config{
		App: AppConfig{
			Env:  v.GetString("APP_ENV"),
			Port: v.GetString("APP_PORT"),
			Name: v.GetString("APP_NAME"),
		},
		Database: dbCfg,
		Redis: RedisConfig{
			Host:     v.GetString("REDIS_HOST"),
			Port:     v.GetString("REDIS_PORT"),
			Password: v.GetString("REDIS_PASSWORD"),
			DB:       v.GetInt("REDIS_DB"),
		},
		JWT: JWTConfig{
			Secret:          v.GetString("JWT_SECRET"),
			AccessTokenTTL:  v.GetInt("JWT_ACCESS_TOKEN_TTL"),
			RefreshTokenTTL: v.GetInt("JWT_REFRESH_TOKEN_TTL"),
		},
		FCM: FCMConfig{
			ServerKey: v.GetString("FCM_SERVER_KEY"),
		},
	}
}
