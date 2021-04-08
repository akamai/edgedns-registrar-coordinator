// Copyright 2021 Akamai Technologies, Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"fmt"
	"time"
)

const (
	DefaultIntervalMinutes = 15
	DefaultInterval        = time.Minute * DefaultIntervalMinutes
)

var (
	DefaultConfig = Config{
		Registrar:             "",
		RegistrarConfigPath:   "",
		Interval:              DefaultInterval,
		EdgeDNSContract:       "",
		EdgeDNSGroup:          0,
		EdgegridHost:          "",
		EdgegridClientToken:   "",
		EdgegridClientSecret:  "",
		EdgegridAccessToken:   "",
		EdgegridEdgercPath:    "",
		EdgegridEdgercSection: "",
		LogFilePath:           "",
		LogHandler:            "text",
		LogLevel:              "info",
		PluginLibPath:         "",
	}
)

// Internal master config. Reflects all accepted command line directives
type Config struct {
	Registrar           string
	RegistrarConfigPath string        // Registrar conffg file path. Parsed by Registrar Provider
	Interval            time.Duration // Default: 15 minutes
	DNSSEC              bool
	TSig                bool
	FailOnError         bool
	// Edge DNS Credentials
	EdgeDNSContract      string
	EdgeDNSGroup         int
	EdgegridHost         string
	EdgegridClientToken  string
	EdgegridClientSecret string
	EdgegridAccessToken  string
	// --OR--
	EdgegridEdgercPath    string
	EdgegridEdgercSection string
	// Optional
	LogFilePath string
	LogHandler  string
	LogLevel    string
	DryRun      bool
	Once        bool
	// Plugin Registrar
	PluginLibPath string
	// Add MarkMonitor ….
}

func NewConfig() *Config {

	return &Config{}

}

func NewApp() *kingpin.Application {

	return kingpin.New("edgedns-registrar-coordinator", "A command-line application for coordination of registrar actions with Akamai Edge DNS.\n\nNote that all flags may be replaced with env vars - `--registrar` -> `EDGEDNS-REGISTRAR-COORDINATOR-REGISTRAR` or `--registrar value` -> `EDGEDNS-REGISTRAR-COORDINATOR-REGISTRAR=value`")

}

// ParseFlags adds and parses flags from command line
func (cfg *Config) ParseFlags(app *kingpin.Application, args []string) (string, error) {

	fmt.Println("ParseFlags Command Args: ", args)
	//app.Version(Version)
	app.DefaultEnvars() // ParseFlags adds and parses flags from command line
	app.Flag("registrar", "registrar").Required().StringVar(&cfg.Registrar)
	app.Flag("registrar-config-path", "registrar configuration filepath").StringVar(&cfg.RegistrarConfigPath)
	app.Flag("interval", "registrar coordination interval in duration format (default: 15m)").Default(DefaultConfig.Interval.String()).DurationVar(&cfg.Interval)
	app.Flag("fail-on-error", "Fail and exit on error during sub command processing").BoolVar(&cfg.FailOnError)
	app.Flag("dnssec", "Enables DNSSEC Serve(default: disabled").BoolVar(&cfg.DNSSEC)
	app.Flag("tsig", "Enables TSIG Key processing (default: disabled").BoolVar(&cfg.TSig)
	app.Flag("once", "When enabled, exits the synchronization loop after the first iteration (default: disabled)").BoolVar(&cfg.Once)
	app.Flag("dry-run", "When enabled, prints DNS record changes rather than actually performing them (default: disabled)").BoolVar(&cfg.DryRun)
	app.Flag("log-file-path", "The log file path. Default destination is stderr ").Default(DefaultConfig.LogFilePath).StringVar(&cfg.LogFilePath)
	app.Flag("log-handler", "The handler used to log messages (default: text. options: text, json, cli, discard)").Default(DefaultConfig.LogHandler).EnumVar(&cfg.LogHandler, "text", "json", "cli", "discard", "syslog")
	app.Flag("log-level", "Set the level of logging. (default: info, options: debug, info, warning, error, fatal").Default(DefaultConfig.LogLevel).EnumVar(&cfg.LogLevel, "debug", "info", "warning", "error", "fatal")

	// Edge DNS
	app.Flag("edgedns-contract", "Contract to use creating a domain.").Default(DefaultConfig.EdgeDNSContract).StringVar(&cfg.EdgeDNSContract)
	app.Flag("edgedns-group", "group id to use creating a domain.").IntVar(&cfg.EdgeDNSGroup)
	app.Flag("edgegrid-host", "EdgeDNS API Server URL.").Default(DefaultConfig.EdgegridHost).StringVar(&cfg.EdgegridHost)
	app.Flag("edgegrid-client-token", "EdgeDNS API Client Token.").Default(DefaultConfig.EdgegridClientToken).StringVar(&cfg.EdgegridClientToken)
	app.Flag("edgegrid-client-secret", "EdgeDNS API Client Secret.").Default(DefaultConfig.EdgegridClientSecret).StringVar(&cfg.EdgegridClientSecret)
	app.Flag("edgegrid-access-token", "EdgeDNS API Access Token.").Default(DefaultConfig.EdgegridAccessToken).StringVar(&cfg.EdgegridAccessToken)
	app.Flag("edgegrid-edgerc-path", "optionally specify the .edgerc file path instead of individual Edgegrid keys").Default(DefaultConfig.EdgegridEdgercPath).StringVar(&cfg.EdgegridEdgercPath)
	app.Flag("edgegrid-edgerc-section", "specify the section when specifying an .edgerc file path").Default(DefaultConfig.EdgegridEdgercSection).StringVar(&cfg.EdgegridEdgercSection)

	// Plugin Registrat Orpovider
	app.Flag("plugin-filepath", "plugin provider library location path.").Default(DefaultConfig.PluginLibPath).StringVar(&cfg.PluginLibPath)

	cmd, err := app.Parse(args)
	if err != nil {
		return cmd, err
	}

	return cmd, nil
}

// Validate config
func (cfg *Config) Validate() error {

	if cfg.Interval <= 0 {
		return fmt.Errorf("Interval must be greter than zero")
	}

	if cfg.Registrar == "" {
		return fmt.Errorf("no registrar specified")
	}

	if cfg.EdgeDNSContract == "" {
		return fmt.Errorf("edgedns contract is required")
	}

	if cfg.EdgeDNSGroup < 1 {
		return fmt.Errorf("edgedns group is required")
	}

	if cfg.EdgegridHost == "" && cfg.EdgegridEdgercPath == "" {
		return fmt.Errorf("no Edgegrid Host specified")
	}
	if cfg.EdgegridClientToken == "" && cfg.EdgegridEdgercPath == "" {
		return fmt.Errorf("no Edgegrid client token specified")
	}
	if cfg.EdgegridClientSecret == "" && cfg.EdgegridEdgercPath == "" {
		return fmt.Errorf("no Edgegrid client secret specified")
	}
	if cfg.EdgegridAccessToken == "" && cfg.EdgegridEdgercPath == "" {
		return fmt.Errorf("no Edgegrid access token specified")
	}

	// Plugin Registrar
	if cfg.Registrar == "plugin" && cfg.PluginLibPath == "" {
		return fmt.Errorf("plugin library filepath must be specified for plugin registrar")
	}

	// All good
	return nil

}
