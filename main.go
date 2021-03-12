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
	log "github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/apex/log/handlers/discard"
	"github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/text"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	//"github.com/akamai/edgedns-registrar-coordinator/registrar/plugin"

	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
)

const ()

var (
	edgeDNSHandler *internal.EdgeDNSHandler
	app            *kingpin.Application
	// monitor sub command
	monitor *kingpin.CmdClause
)

func main() {

	var err error

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
			cancel()
		case <-ctx.Done():
		}
		<-signalChan
		os.Exit(2)
	}()

	cfg := internal.NewConfig()
	app = internal.NewApp()
	monitor = app.Command("monitor", "Monitor registrar for domain adds and deletes.")
	if len(os.Args) < 2 {
		app.FatalUsage("/nError: sub command is required/n")
	}
	cmd, err := cfg.ParseFlags(app, os.Args[1:])
	if err != nil {
		fmt.Println("flag parsing error: ", err.Error())
		app.FatalUsage("command line parsing error: %v", err.Error())
	}
	err = cfg.Validate()
	if err != nil {
		fmt.Println("validation error: ", err.Error())
		app.FatalUsage("command line validation error: %v", err.Error())
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

	default:
		log.Warn("Log handler invalid. Using default text handler")
		log.SetHandler(text.Default)
	}

	monitor = app.Command("monitor", "Monitor registrar for domain adds and deletes.")
	if err != nil {
		log.Fatalf("sub command parse error: %v", err)
		app.FatalUsage("command line validation error: %v", err.Error())
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
				AkamaiContracts:     strings.Join(cfg.AkamaiContracts, ","),
				AkamaiConfigPath:    cfg.RegistrarConfigPath,
				Interval:            cfg.Interval,
				AkamaiNameFilter:    cfg.AkamaiNameFilter,
				AkamaiHost:          cfg.AkamaiHost,
				AkamaiClientToken:   cfg.AkamaiClientToken,
				AkamaiClientSecret:  cfg.AkamaiClientSecret,
				AkamaiAccessToken:   cfg.AkamaiAccessToken,
				AkamaiEdgercPath:    cfg.AkamaiEdgercPath,
				AkamaiEdgercSection: cfg.AkamaiEdgercSection,
				Once:                cfg.Once,
				DryRun:              cfg.DryRun,
			},
			nil,
		)
		/*
		   	case "plugin":
		   		r, err = akamai.NewPluginRegistrar(
		                           plugin.PluginConfig{
		                                   PluginLibPath   cfg.PluginLibPath,
		                                   Once:           cfg.Once,
		                                   DryRun:         cfg.DryRun,
		                           },
		                   )
		*/
	default:
		err = fmt.Errorf("Invalid command")
	}
	if err != nil {
		appLog.Errorf("Failed to create registrar. Error: %s", err.Error())
		app.Fatalf("Failed to create registrar. Error: %s", err.Error())
	}
	//log.AddFlags(kingpin.CommandLine)
	/*
	   app.Version(version.Print("edgedns_registrar_coordinator"))
	   log.Info("Starting Edge DNS Registrar Coordinator", version.Info())
	   log.Info("Build context", version.BuildContext())
	*/

	// Init EdgeDNSHandler
	edgeDNSHandler, err = internal.InitEdgeDNSHandler(ctx, cfg, nil)
	if err != nil {
		appLog.Errorf("Failed to initialize Edge DNS Handler. Error: %s", err.Error())
		app.Fatalf("Failed to initialize Edge DNS Handler. Error: %s", err.Error())
	}

	// Pass back error message ...
	cmderr := make(chan string)

	switch cmd {
	case monitor.FullCommand():
		appLog.Info("Processing monitor command")
		go internal.Monitor(ctx, cmderr, cfg.Registrar, r, edgeDNSHandler, cfg.Interval, cfg.DryRun, cfg.Once)

	default:
		appLog.Errorf("Invalid commandline [%s]", strings.Join(os.Args, " "))
		app.FatalUsage("Invalid commandline [%s]", strings.Join(os.Args, " "))
	}

	errmsg := <-cmderr
	if errmsg != "" {
		appLog.Errorf("Command action failed. Error: %s", errmsg)
		app.Fatalf("Command action failed. Error: %s", errmsg)
	}
}
