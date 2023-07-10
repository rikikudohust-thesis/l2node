package builder

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/kelseyhightower/envconfig"
	"github.com/mcuadros/go-defaults"
	"github.com/spf13/viper"

	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"
)

type jobConfig struct {
	Config   model.JobConfig
	Database dbConfig
	Redis    redisConfig
}

func loadJobConfig(file string) (*jobConfig, error) {
	viper.SetConfigFile(file)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	c := &jobConfig{}
	defaults.SetDefaults(c)
	if err := viper.Unmarshal(c); err != nil {
		return nil, err
	}

	dbConfig, redisCfg, err := LoadEnvConfig()
	if err != nil {
		return nil, err
	}
	c.Database = *dbConfig
	c.Redis = *redisCfg
	c.Config.Contracts.ZKPayment = common.HexToAddress(c.Config.ZKPayment)

	fmt.Printf("config: %+v\n", c.Config)
	return c, nil
}

func LoadEnvConfig() (*dbConfig, *redisConfig, error) {
	var dbCfg dbConfig
	if err := envconfig.Process("DB", &dbCfg); err != nil {
		return nil, nil, err
	}

	var redisCfg redisConfig
	if err := envconfig.Process("REDIS", &redisCfg); err != nil {
		return nil, nil, err
	}

	return &dbCfg, &redisCfg, nil
}
