package config

import (
	"crypto/tls"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v10"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var errConfig = errors.New("config error")

func configError(msg string) error {
	return fmt.Errorf("%w: %s", errConfig, msg)
}

type Config struct {
	Addr             string `env:"GRPC_ADDR" envDefault:"grpc.buenavista.wardenprotocol.org:443" mapstructure:"GRPC_ADDR"`
	EnvFile          string `env:"ENV_FILE" envDefault:""`
	Port             string `env:"PORT" envDefault:"8081" mapstructure:"PORT"`
	TLS              bool   `env:"GRPC_TLS_ENABLED" envDefault:"true" mapstructure:"GRPC_TLS_ENABLED"`
	Timeout          int    `env:"GRPC_TIMEOUT_SECONDS" envDefault:"45" mapstructure:"GRPC_TIMEOUT_SECONDS"`
	TTL              int    `env:"TTL" envDefault:"60" mapstructure:"TTL"`
	ChainID          string `env:"CHAIN_ID" envDefault:"buenavista-1" mapstructure:"CHAIN_ID"`
	WardenMetrics    bool   `env:"WARDEN_METRICS" envDefault:"true" mapstructure:"WARDEN_METRICS"`
	ValidatorMetrics bool   `env:"VALIDATOR_METRICS" envDefault:"true" mapstructure:"VALIDATOR_METRICS"`
	WalletAddresses  string `env:"WALLET_ADDRESSES" envDefault:"" mapstructure:"WALLET_ADDRESSES"`
	Denom            string `env:"DENOM" envDefault:"uward" mapstructure:"DENOM"`
	WarpMetrics      bool   `env:"WARP_METRICS" envDefault:"true" mapstructure:"WARP_METRICS"`
	WarpDB           string `env:"WARP_DATABASE" envDefault:"" mapstructure:"WARP_DATABASE"`
	WarpDBUser       string `env:"WARP_DATABASE_USER" envDefault:"" mapstructure:"WARP_DATABASE_USER"`
	WarpDBPass       string `env:"WARP_DATABASE_PASS" envDefault:"" mapstructure:"WARP_DATABASE_PASS"`
	WarpDBHost       string `env:"WARP_DATABASE_HOST" envDefault:"" mapstructure:"WARP_DATABASE_HOST"`
}

func LoadConfig() (Config, error) {
	cfg := Config{}
	var err error

	// setDefaults(*cfg)

	if err = env.Parse(&cfg); err != nil {
		return Config{}, configError(err.Error())
	}

	if cfg.EnvFile != "" {
		if err = loadConfigFile(&cfg); err != nil {
			return Config{}, configError(err.Error())
		}
	}
	return cfg, nil
}

func loadConfigFile(cfg *Config) error {
	var err error

	// parse config file params
	// Extract the directory
	dir := filepath.Dir(cfg.EnvFile) + "/"

	// Extract the base name (filename without directory)
	base := filepath.Base(cfg.EnvFile)

	// Split the base name into name and extension
	name := strings.TrimSuffix(base, filepath.Ext(base))
	ext := strings.TrimPrefix(filepath.Ext(base), ".")

	viper.AddConfigPath(dir)
	viper.SetConfigName(name)
	viper.SetConfigType(ext)

	viper.AutomaticEnv()
	err = viper.ReadInConfig()
	if err != nil {
		return err
	}

	if err = viper.Unmarshal(&cfg); err != nil {
		return err
	}
	return nil
}

func (c Config) GRPCConn() (*grpc.ClientConn, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	transportCreds := grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))

	if !c.TLS {
		transportCreds = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	conn, err := grpc.Dial(
		c.Addr,
		transportCreds,
		grpc.WithDefaultCallOptions(grpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
	)
	if err != nil {
		return nil, configError(err.Error())
	}

	return conn, nil
}
