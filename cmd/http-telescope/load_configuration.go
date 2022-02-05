package main

import (
	"errors"
	"fmt"
	"github.com/ardanlabs/conf"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"time"
)

type LogSettings struct {
	Level       string `conf:"default:warn"`
	MethodName  bool   `conf:"default:false"`
	JSON        bool   `conf:"default:false"`
	Destination string `conf:"default:stderr"`
	File        string `conf:"default:/tmp/debug.log"`
}

type Configuration struct {
	Config struct {
		Path string `conf:"default:/conf/config.yml"`
	}
	Web struct {
		Listen          string        `conf:"default:0.0.0.0:3001"`
		UI              string        `conf:"default:0.0.0.0:3002"`
		ReadTimeout     time.Duration `conf:"default:5s"`
		WriteTimeout    time.Duration `conf:"default:5s"`
		ShutdownTimeout time.Duration `conf:"default:5s"`
		Upstream        string        `conf:""`
	}
	Log            LogSettings
	DisableCaching bool `conf:"default:false"`
	StoreBody      bool `conf:"default:false"`
}

func loadConfiguration() (Configuration, error) {
	// Create configuration defaults
	var cfg Configuration

	// Try to load configuration from environment variables and command line switches
	if err := conf.Parse(os.Args[1:], "CFG", &cfg); err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			usage, err := conf.Usage("CFG", &cfg)
			if err != nil {
				return cfg, fmt.Errorf("generating config usage: %w", err)
			}
			fmt.Println(usage) //nolint:forbidigo
			return cfg, conf.ErrHelpWanted
		}
		return cfg, fmt.Errorf("parsing config: %w", err)
	}

	// Override values from YAML if specified and if it exists (useful in k8s/compose)
	fp, err := os.Open(cfg.Config.Path)
	if err != nil && !os.IsNotExist(err) {
		return cfg, fmt.Errorf("can't read the config file, while it exists: %w", err)
	} else if err == nil {
		yamlFile, err := ioutil.ReadAll(fp)
		if err != nil {
			return cfg, fmt.Errorf("can't read config file: %w", err)
		}
		err = yaml.Unmarshal(yamlFile, &cfg)
		if err != nil {
			return cfg, fmt.Errorf("can't unmarshal config file: %w", err)
		}
		_ = fp.Close()
	}

	return cfg, nil
}

// newLogger istantiates a new logging backend
func newLogger(cfg LogSettings) (logrus.FieldLogger, error) {
	// Init Logging
	logger := logrus.New()

	// Setting output
	switch {
	case cfg.Destination == "stdout":
		logger.SetOutput(os.Stdout)
	case cfg.Destination == "stderr":
		logger.SetOutput(os.Stderr)
	case cfg.Destination == "file":
		file, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err == nil {
			logger.SetOutput(file)
		} else {
			logger.SetOutput(os.Stderr)
			logger.WithError(err).Error("Can't open log file for writing, using stderr")
		}
	}

	// Set logging level
	switch cfg.Level {
	case "trace":
		logger.SetLevel(logrus.TraceLevel)
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	case "info":
		fallthrough
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	if cfg.MethodName {
		logger.SetReportCaller(true)
	}

	if cfg.JSON {
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return logger.WithFields(logrus.Fields{
		"hostname": hostname,
	}), nil
}
