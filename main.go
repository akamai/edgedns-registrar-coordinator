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

package main

import (
	"github.com/akamai/edgedns-registrar-coordinator/internal"
	"github.com/akamai/edgedns-registrar-coordinator/registrar"
	akamai "github.com/akamai/edgedns-registrar-coordinator/registrar/akamai"
	markmonitorsftp "github.com/akamai/edgedns-registrar-coordinator/registrar/markmonitorsftp"
	plugin "github.com/akamai/edgedns-registrar-coordinator/registrar/plugin"
	log "github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/apex/log/handlers/discard"
	"github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/text"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
)

const ()

var (
	// Application version
	VERSION        = "0.1.0"
	edgeDNSHandler *internal.EdgeDNSHandler
	app            *kingpin.Application
	// monitor sub command
	monitor *kingpin.CmdClause
)

func main() {

	var err error
	cmderr := make(chan string)

	// create a context
	ctx := context.Background()

	// Setup to catch ctl-C. Add logger later ...
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	go func() {
		select {
		case <-signalChan:
			fmt.Println("SIGNAL")
			cmderr <- "Interrupt signal received"
			cancel()
		case <-ctx.Done():
		}
		<-signalChan
		os.Exit(0)
	}()

	cfg := internal.NewConfig()
	app = internal.NewApp()
	monitor = app.Command("monitor", "Monitor registrar for domain adds and deletes.")
	if len(os.Args) < 2 {
		app.FatalUsage("/nError: sub command is required/n")
		os.Exit(1)
	}
	cmd, err := cfg.ParseFlags(app, os.Args[1:])
	if err != nil {
		fmt.Println("flag parsing error: ", err.Error())
		app.FatalUsage("command line parsing error: %v", err.Error())
		os.Exit(1)
	}
	err = cfg.Validate()
	if err != nil {
		fmt.Println("validation error: ", err.Error())
		app.FatalUsage("command line validation error: %v", err.Error())
		os.Exit(1)
	}

	// Setup logging
	logLevel, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Warnf("Log level invalid. Using default, %s", internal.DefaultConfig.LogLevel)
		logLevel, err = log.ParseLevel(internal.DefaultConfig.LogLevel)
	}
	log.SetLevel(logLevel)
	out := os.Stderr
	if cfg.LogFilePath != "" {
		f, err := os.OpenFile(cfg.LogFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Println("Could Not Open Log File : " + err.Error())
		}
		out = f
	}
	switch cfg.LogHandler {
	case "cli":
		log.SetHandler(cli.Default)

	case "text":
		log.SetHandler(text.New(out))

	case "json":
		log.SetHandler(json.New(out))

	case "discard":
		log.SetHandler(discard.Default)

	/*
			case "syslog":
		                levelMap := map[string]string{
		                        "debug": ,
					"info": ,
					"warning": ,
					"error": ,
					"fatal": ,
		                }
	*/

	default:
		log.Warn("Log handler invalid. Using default text handler")
		log.SetHandler(text.Default)
	}

	monitor = app.Command("monitor", "Monitor registrar for domain adds and deletes.")
	if err != nil {
		log.Fatalf("sub command parse error: %v", err)
		app.FatalUsage("command line validation error: %v", err.Error())
		os.Exit(1)
	}

	appLog := log.WithFields(log.Fields{
		"registrar":  cfg.Registrar,
		"subcommand": cmd,
	})

	ctx = context.WithValue(ctx, "appLog", appLog)
	// Initialize registrar provider
	var r registrar.RegistrarProvider
	switch cfg.Registrar {
	case "akamai":
		r, err = akamai.NewAkamaiRegistrar(
			ctx,
			akamai.AkamaiConfig{
				AkamaiConfigPath: cfg.RegistrarConfigPath,
			},
			nil,
		)

	case "plugin":
		appLog.Debugf("PluginLibPath: %s", cfg.PluginLibPath)
		appLog.Debugf("PluginConfigPath: %s", cfg.RegistrarConfigPath)
		r, err = plugin.NewPluginRegistrar(
			ctx,
			registrar.PluginConfig{
				PluginLibPath:    cfg.PluginLibPath,
				PluginConfigPath: cfg.RegistrarConfigPath,
				LogEntry:         appLog,
			},
		)

	case "markmonitorsftp":
		r, err = markmonitorsftp.NewMarkMonitorSFTPRegistrar(
			ctx,
			markmonitorsftp.MarkMonitorSFTPConfig{
				MarkMonitorSFTPConfigPath: cfg.RegistrarConfigPath,
			},
			nil,
		)

	default:
		err = fmt.Errorf("Invalid command")
	}
	if err != nil {
		appLog.Errorf("Failed to create registrar. Error: %s", err.Error())
		app.Fatalf("Failed to create registrar. Error: %s", err.Error())
		os.Exit(1)
	}

	app.Version((VERSION))
	log.Infof("Starting Edge DNS Registrar Coordinator version %s", VERSION)

	// Init EdgeDNSHandler
	edgeDNSHandler, err = internal.InitEdgeDNSHandler(ctx, cfg, nil)
	if err != nil {
		appLog.Errorf("Failed to initialize Edge DNS Handler. Error: %s", err.Error())
		app.Fatalf("Failed to initialize Edge DNS Handler. Error: %s", err.Error())
		os.Exit(1)
	}

	switch cmd {
	case monitor.FullCommand():
		appLog.Info("Processing monitor command")
		go internal.Monitor(ctx, cmderr, cfg.Registrar, r, edgeDNSHandler, cfg.Interval, cfg.DryRun, cfg.Once)

	default:
		appLog.Errorf("Invalid commandline [%s]", strings.Join(os.Args, " "))
		app.FatalUsage("Invalid commandline [%s]", strings.Join(os.Args, " "))
		os.Exit(1)
	}
	errmsg := <-cmderr
	if errmsg != "" {
		if strings.Contains(errmsg, "Interrupt") {
			appLog.Infof("Command action terminated. %s", errmsg)
			fmt.Println("Command action terminated. ", errmsg)
		} else {
			appLog.Errorf("Command action terminated. %s", errmsg)
			app.Fatalf("Command action terminated. %s", errmsg)
			os.Exit(1)
		}
	}
}
