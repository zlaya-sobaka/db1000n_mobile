// Package utils [general utility functions for the db1000n app]
package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/zlaya-sobaka/db1000n_mobile/src/mobilelogger"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// PanicHandler just stub it in the beginning of every major module invocation to prevent single module failure from crashing the whole app
func PanicHandler(logger *zap.Logger) {
	if err := recover(); err != nil {
		logger.Error("caught panic", zap.Any("err", err))
	}
}

// GetEnvStringDefault returns environment variable or default value if no env varible is present
func GetEnvStringDefault(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	return value
}

// GetEnvIntDefault returns environment variable or default value if no env varible is present
func GetEnvIntDefault(key string, defaultValue int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	v, err := strconv.Atoi(value)
	if err != nil {
		mobilelogger.Infof("GetEnvIntDefault[%s]: %v", key, err)

		return defaultValue
	}

	return v
}

// GetEnvBoolDefault returns environment variable or default value if no env varible is present
func GetEnvBoolDefault(key string, defaultValue bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	v, err := strconv.ParseBool(value)
	if err != nil {
		mobilelogger.Infof("GetEnvBoolDefault[%s]: %v", key, err)

		return defaultValue
	}

	return v
}

// GetEnvDurationDefault returns environment variable or default value if no env varible is present
func GetEnvDurationDefault(key string, defaultValue time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	v, err := time.ParseDuration(value)
	if err != nil {
		mobilelogger.Infof("GetEnvBoolDefault[%s]: %v", key, err)

		return defaultValue
	}

	return v
}

func NonNilDurationOrDefault(d *time.Duration, dflt time.Duration) time.Duration {
	if d != nil {
		return *d
	}

	return dflt
}

func NonNilIntOrDefault(i *int, dflt int) int {
	if i != nil {
		return *i
	}

	return dflt
}

// Decode is an alias to a mapstructure.NewDecoder({Squash: true}).Decode()
func Decode(input interface{}, output interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{Squash: true, WeaklyTypedInput: true, Result: output})
	if err != nil {
		mobilelogger.Infof("Error parsing job config: %v", err)

		return err
	}

	return decoder.Decode(input)
}

func Unmarshal(input []byte, output interface{}, format string) error {
	switch format {
	case "", "json", "yaml":
		if err := yaml.Unmarshal(input, output); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown config format: %v", format)
	}

	return nil
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		_ = exec.Command("open", url).Start()
	}

	mobilelogger.Infof("Please open %s", url)
}
