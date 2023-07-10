package context

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/logging"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var config struct {
	LogLevel      uint32          `default:"4" split_words:"true"`
	stdoutLogger  *logrus.Logger  `ignored:"true"`
	sdClient      *logging.Client `ignored:"true"`
	sdLogger      *logging.Logger `ignored:"true"`
	sdEventLogger *logging.Logger `ignored:"true"`
}

type googleConfig struct {
	Project      string `split_words:"true"`
	LogName      string `split_words:"true"`
	EventLogName string `split_words:"true"`
}

func init() {
	if err := envconfig.Process("CONTEXT", &config); err != nil {
		log.Fatalf("failed to parse context config, err=%s", err.Error())
	}

	config.stdoutLogger = &logrus.Logger{
		Out: os.Stdout,
		Formatter: &prefixed.TextFormatter{
			FullTimestamp:   true,
			ForceFormatting: true,
			ForceColors:     true,
		},
		Level: logrus.Level(config.LogLevel),
	}

	var gConfig googleConfig
	if err := envconfig.Process("GOOGLE_CLOUD", &gConfig); err != nil {
		log.Fatalf("failed to parse google cloud config, err=%s", err.Error())
	}

	if gConfig.Project != "" {
		var err error
		config.sdClient, err = logging.NewClient(context.Background(), gConfig.Project)
		if err != nil {
			config.stdoutLogger.Errorf("failed to init google cloud logging client: %v", err)
		}
	}

	if gConfig.LogName != "" && config.sdClient != nil {
		config.sdLogger = config.sdClient.Logger(gConfig.LogName)
	}

	if gConfig.EventLogName != "" && config.sdClient != nil {
		config.sdEventLogger = config.sdClient.Logger(gConfig.EventLogName)
	}
}
