package config

import (
	"crypto/tls"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

var errConfig = errors.New("config error")

func configError(msg string) error {
	return fmt.Errorf("%w: %s", errConfig, msg)
}

//nolint:lll //this struct cannot be changed to smaller one
type Config struct {
	GRPCAddr         string `env:"GRPC_ADDR"            envDefault:"grpc.wardenprotocol.org:443" mapstructure:"GRPC_ADDR"`
	EnvFile          string `env:"ENV_FILE"             envDefault:""`
	Port             string `env:"PORT"                 envDefault:"8081"                        mapstructure:"PORT"`
	TLS              bool   `env:"GRPC_TLS_ENABLED"     envDefault:"true"                        mapstructure:"GRPC_TLS_ENABLED"`
	Timeout          int    `env:"GRPC_TIMEOUT_SECONDS" envDefault:"45"                          mapstructure:"GRPC_TIMEOUT_SECONDS"`
	TTL              int    `env:"TTL"                  envDefault:"60"                          mapstructure:"TTL"`
	ChainID          string `env:"CHAIN_ID"             envDefault:"warden_8765-1"               mapstructure:"CHAIN_ID"`
	ValidatorMetrics bool   `env:"VALIDATOR_METRICS"    envDefault:"true"                        mapstructure:"VALIDATOR_METRICS"`
	WalletAddresses  string `env:"WALLET_ADDRESSES"     envDefault:""                            mapstructure:"WALLET_ADDRESSES"`
	Denom            string `env:"DENOM"                envDefault:"award"                       mapstructure:"DENOM"`
	Exponent         int    `env:"EXPONENT"             envDefault:"18"                          mapstructure:"EXPONENT"`
	VeniceMetrics    bool   `env:"VENICE_METRICS"       envDefault:"false"                       mapstructure:"VENICE_METRICS"`
	VeniceAPIKey     string `env:"VENICE_API_KEY"       envDefault:""                            mapstructure:"VENICE_API_KEY"`
	MessariMetrics   bool   `env:"MESSARI_METRICS"      envDefault:"false"                       mapstructure:"MESSARI_METRICS"`
	MessariAPIKey    string `env:"MESSARI_API_KEY"      envDefault:""                            mapstructure:"MESSARI_API_KEY"`
	BaseMetrics      bool   `env:"BASE_METRICS"         envDefault:"false"                       mapstructure:"BASE_METRICS"`
	BaseRPCURL       string `env:"BASE_RPC_URL"         envDefault:""                            mapstructure:"BASE_RPC_URL"`
	BaseAddresses    string `env:"BASE_ADDRESSES"       envDefault:""                            mapstructure:"BASE_ADDRESSES"`
	BnbMetrics       bool   `env:"BNB_METRICS"          envDefault:"false"                       mapstructure:"BNB_METRICS"`
	BnbRPCURL        string `env:"BNB_RPC_URL"          envDefault:""                            mapstructure:"BNB_RPC_URL"`
	BnbAddresses     string `env:"BNB_ADDRESSES"        envDefault:""                            mapstructure:"BNB_ADDRESSES"`
	CoinGeckoMetrics bool   `env:"COINGECKO_METRICS"    envDefault:"false"                       mapstructure:"COINGECKO_METRICS"`
	CoinGeckoAPIKey  string `env:"COINGECKO_API_KEY"    envDefault:""                            mapstructure:"COINGECKO_API_KEY"`
	XAIMetrics       bool   `env:"XAI_METRICS"          envDefault:"false"                       mapstructure:"XAI_METRICS"`
	XAIAPIKey        string `env:"XAI_API_KEY"          envDefault:""                            mapstructure:"XAI_API_KEY"`
	XAITeamID        string `env:"XAI_TEAM_ID"          envDefault:""                            mapstructure:"XAI_TEAM_ID"`
	OpenAIMetrics    bool   `env:"OPENAI_METRICS"       envDefault:"false"                       mapstructure:"OPENAI_METRICS"`
	OpenAIAPIKey     string `env:"OPENAI_API_KEY"       envDefault:""                            mapstructure:"OPENAI_API_KEY"`
	BlockWindow      int64  `env:"BLOCK_WINDOW"         envDefault:"200"                         mapstructure:"BLOCK_WINDOW"`
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

	conn, err := grpc.NewClient(
		c.GRPCAddr,
		transportCreds,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt64),
			grpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Duration(c.Timeout),
			Timeout:             time.Duration(c.Timeout),
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return nil, configError(err.Error())
	}

	return conn, nil
}
