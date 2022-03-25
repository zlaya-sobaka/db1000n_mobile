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

// Package config [used for configuring the package]
package config

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/zlaya-sobaka/db1000n_mobile/src/jobs"
	"github.com/zlaya-sobaka/db1000n_mobile/src/mobilelogger"
	"github.com/zlaya-sobaka/db1000n_mobile/src/utils"
)

// Config for all jobs to run
type Config struct {
	Jobs []jobs.Config
}

type RawConfig struct {
	Body         []byte
	Encrypted    bool
	lastModified string
	etag         string
}

// fetch tries to read a config from the list of mirrors until it succeeds
func fetch(paths []string, lastKnownConfig *RawConfig) *RawConfig {
	for i := range paths {
		config, err := fetchSingle(paths[i], lastKnownConfig)
		if err != nil {
			mobilelogger.Infof("Failed to fetch config from %q: %v", paths[i], err)

			continue
		}

		mobilelogger.Infof("Loading config from %q", paths[i])

		return config
	}

	return lastKnownConfig
}

// fetchSingle reads a config from a single source
func fetchSingle(path string, lastKnownConfig *RawConfig) (*RawConfig, error) {
	configURL, err := url.ParseRequestURI(path)
	// absolute paths can be interpreted as a URL with no schema, need to check for that explicitly
	if err != nil || filepath.IsAbs(path) {
		res, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		return &RawConfig{Body: res, lastModified: "", etag: ""}, nil
	}

	const requestTimeout = 20 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, configURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("If-None-Match", lastKnownConfig.etag)
	req.Header.Add("If-Modified-Since", lastKnownConfig.lastModified)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return lastKnownConfig, nil
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("error fetching config, code %d", resp.StatusCode)
	}

	etag := resp.Header.Get("etag")
	lastModified := resp.Header.Get("last-modified")

	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &RawConfig{Body: res, etag: etag, lastModified: lastModified}, nil
}

// FetchRawConfig retrieves the current config using a list of paths. Falls back to the last known config in case of errors.
func FetchRawConfig(paths []string, lastKnownConfig *RawConfig) *RawConfig {
	newConfig := fetch(paths, lastKnownConfig)

	if utils.IsEncrypted(newConfig.Body) {
		decryptedConfig, err := utils.Decrypt(newConfig.Body)
		if err != nil {
			mobilelogger.Infof("Can't decrypt config")

			return lastKnownConfig
		}

		mobilelogger.Infof("Decrypted config")

		newConfig.Body = decryptedConfig
		newConfig.Encrypted = true
	}

	return newConfig
}

// Unmarshal config encoded with the given format.
func Unmarshal(body []byte, format string) *Config {
	if body == nil {
		return nil
	}

	var config Config

	if err := utils.Unmarshal(body, &config, format); err != nil {
		return nil
	}

	return &config
}
