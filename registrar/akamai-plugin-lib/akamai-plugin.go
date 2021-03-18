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
	log "github.com/apex/log"

	"github.com/akamai/edgedns-registrar-coordinator/registrar"

	"fmt"
	dns "github.com/akamai/AkamaiOPEN-edgegrid-golang/configdns-v2"
	edgegrid "github.com/akamai/AkamaiOPEN-edgegrid-golang/edgegrid"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const ()

var (
	akamaiLibRegistrar AkamaiLibRegistrar
	LibPluginArgs      registrar.PluginFuncArgs
	LibPluginResult    registrar.PluginFuncResult
	libLog             *log.Entry
)

// edgeDNSClient is a proxy interface of the Akamai edgegrid configdns-v2 package that can be stubbed for testing.
type AkamaiDNSService interface {
	GetDomains()
	GetDomain(domain string)
	GetTsigKey(domain string)
	GetServeAlgorithm(domain string)
	GetMasterIPs()
}

// AkamaiProvider implements the DNS provider for Akamai.
type AkamaiLibRegistrar struct {
	//registrar.BaseRegistrarProvider
	akaConfig *AkamaiConfig
	config    *edgegrid.Config
	// Defines client. Allows for mocking.
	client AkamaiDNSService
}

type AkamaiConfig struct {
	AkamaiContracts     string `yaml:"akamai_contracts"`
	AkamaiNameFilter    string `yaml:"akamai_name_filter"`
	AkamaiEdgercPath    string `yaml:"akamai_edgerc_path"`
	AkamaiEdgercSection string `yaml:"akamai_edgerc_section"`
	AkamaiHost          string `yaml:"akamai_host"`
	AkamaiAccessToken   string `yaml:"akamai_access_token"`
	AkamaiClientToken   string `yaml:"akamai_client_token"`
	AkamaiClientSecret  string `yaml:"akamai_client_secret"`
	MaxBody             int    `yaml:"akamai_client_maxbody"`
	AccountKey          string `yaml:"akamai_client_account_key"`
}

// NewAkamaiProvider initializes a new Akamai DNS based Provider.
func NewPluginLibRegistrar() {

	var akaConfig *AkamaiConfig
	var err error

	pluginConfig := LibPluginArgs.PluginArg.(registrar.PluginConfig)
	libLog = pluginConfig.LogEntry
	// Get file config and parse
	if pluginConfig.PluginConfigPath == "" {
		LibPluginResult.PluginError = fmt.Errorf("Akamai Plugin Library requires a configurtion file")
		return
	}
	akaConfig, err = loadConfig(pluginConfig.PluginConfigPath)
	if err != nil {
		LibPluginResult.PluginError = fmt.Errorf("Akamai Plugin Library configuration file load failed. Error: %s", err.Error())
		return
	}

	// validate config
	if len(akaConfig.AkamaiContracts) == 0 {
		LibPluginResult.PluginError = fmt.Errorf("Akamai Plugin Library configuration requires one or more contracts.")
		return
	}
	if akaConfig.AkamaiEdgercPath == "" && (akaConfig.AkamaiHost == "" || akaConfig.AkamaiAccessToken == "" || akaConfig.AkamaiClientToken == "" || akaConfig.AkamaiClientSecret == "") {
		LibPluginResult.PluginError = fmt.Errorf("Akamai Plugin Library configuration requires valid set of auth keys.")
		return
	}
	// Process creds. Could be on cmd line, config file
	var edgeGridConfig edgegrid.Config

	libLog.Debugf("Host: %s", akaConfig.AkamaiHost)
	libLog.Debugf("ClientToken: %s", akaConfig.AkamaiClientToken)
	libLog.Debugf("ClientSecret: %s", akaConfig.AkamaiClientSecret)
	libLog.Debugf("AccessToken: %s", akaConfig.AkamaiAccessToken)
	libLog.Debugf("EdgePath: %s", akaConfig.AkamaiEdgercPath)
	libLog.Debugf("EdgeSection: %s", akaConfig.AkamaiEdgercSection)
	libLog.Debugf("AkamaiContracts: %v", akaConfig.AkamaiContracts)
	// environment overrides edgerc file but config needs to be complete
	if akaConfig.AkamaiHost == "" || akaConfig.AkamaiClientToken == "" || akaConfig.AkamaiClientSecret == "" || akaConfig.AkamaiAccessToken == "" {
		// Look for Akamai environment or .edgerd creds
		var err error
		edgeGridConfig, err = edgegrid.Init(akaConfig.AkamaiEdgercPath, akaConfig.AkamaiEdgercSection) // use default .edgerc location and section
		if err != nil {
			libLog.Errorf("Edgegrid Init Failed")
			LibPluginResult.PluginError = err
			return // return empty provider for backward compatibility
		}
	} else {
		// Use external-dns config
		edgeGridConfig = edgegrid.Config{
			Host:         akaConfig.AkamaiHost,
			ClientToken:  akaConfig.AkamaiClientToken,
			ClientSecret: akaConfig.AkamaiClientSecret,
			AccessToken:  akaConfig.AkamaiAccessToken,
			MaxBody:      131072, // same default val as used by Edgegrid
			Debug:        false,
		}
		// Check for edgegrid overrides
		if envval, ok := os.LookupEnv("AKAMAI_MAX_BODY"); ok {
			if i, err := strconv.Atoi(envval); err == nil {
				edgeGridConfig.MaxBody = i
				libLog.Debugf("Edgegrid maxbody set to %s", envval)
			}
		}
		if envval, ok := os.LookupEnv("AKAMAI_ACCOUNT_KEY"); ok {
			edgeGridConfig.AccountKey = envval
			libLog.Debugf("Edgegrid applying account key %s", envval)
		}
		if envval, ok := os.LookupEnv("AKAMAI_DEBUG"); ok {
			if dbgval, err := strconv.ParseBool(envval); err == nil {
				edgeGridConfig.Debug = dbgval
				libLog.Debugf("Edgegrid debug set to %s", envval)
			}
		}
	}

	akamaiLibRegistrar = AkamaiLibRegistrar{
		config:    &edgeGridConfig,
		akaConfig: akaConfig,
	}
	/*
		if akaService != nil {
			log.Debugf("Using STUB")
			provider.client = akaService
		} else {
			provider.client = provider
		}
	*/

	// Init library for direct endpoint calls
	dns.Init(edgeGridConfig)

	return
}

func resetDNSConfig(orig edgegrid.Config) {

	dns.Config = orig

}

func GetDomains() {

	libLog.Debug("Entering Plugin Lib Akamai registrar GetDomains")

	// both edgedns and this plugin using dns. need to temp swap config...
	existConfig := dns.Config
	defer resetDNSConfig(existConfig)
	dns.Config = *akamaiLibRegistrar.config

	LibPluginResult.PluginResult = []string{}

	queryArgs := dns.ZoneListQueryArgs{
		Types:       "PRIMARY",
		SortBy:      "zone",
		ContractIds: akamaiLibRegistrar.akaConfig.AkamaiContracts,
		Search:      akamaiLibRegistrar.akaConfig.AkamaiNameFilter,
	}
	libLog.Debugf("ListZones Query Args: %v", queryArgs)
	zlResp, err := dns.ListZones(queryArgs)
	if err != nil {
		libLog.Debugf("Plugin Lib Registrar GetDomains failed. Error: %s", err.Error())
		LibPluginResult.PluginError = err
		return
	}
	filter := "LOCKED"
	domains := make([]string, 0, len(zlResp.Zones))
	for _, zone := range zlResp.Zones {
		if strings.Contains(filter, zone.ActivationState) {
			continue
		}
		domains = append(domains, zone.Zone)
	}

	libLog.Debugf("Plugin Akamai Registrar GetDomains result: %v", domains)

	LibPluginResult.PluginResult = domains
	return
}

func GetDomain() {
	libLog.Debug("Entering Akamai Plugin Lib registrar GetDomain")

	// both edgedns and this plugin using dns. need to temp swap config...
	existConfig := dns.Config
	defer resetDNSConfig(existConfig)
	dns.Config = *akamaiLibRegistrar.config

	domain := LibPluginArgs.PluginArg.(string)
	zone, err := dns.GetZone(domain)
	if err != nil {
		LibPluginResult.PluginResult = registrar.Domain{}
		LibPluginResult.PluginError = err
		return
	}
	libLog.Debugf("Akamai Plugin Lib Registrar GetDomain result: %v", zone)
	LibPluginResult.PluginResult = registrar.Domain{
		Name:                  zone.Zone,
		Type:                  zone.Type,
		SignAndServe:          zone.SignAndServe,
		SignAndServeAlgorithm: zone.SignAndServeAlgorithm,
		Masters:               zone.Masters,
		TsigKey:               zone.TsigKey,
	}

	return
}

func GetTsigKey() {
	//(tsigKey *dns.TSIGKey, err error) {

	libLog.Debug("Entering Akamai Plugin Lib registrar GetTsigKey")

	// both edgedns and this plugin using dns. need to temp swap config...
	existConfig := dns.Config
	defer resetDNSConfig(existConfig)
	dns.Config = *akamaiLibRegistrar.config

	domain := LibPluginArgs.PluginArg.(string)
	resp, err := dns.GetZoneKey(domain)
	if err != nil {
		LibPluginResult.PluginResult = dns.TSIGKey{}
		LibPluginResult.PluginError = err
		return
	}

	libLog.Debugf("Returning Registrar GetTsigKey result")
	LibPluginResult.PluginResult = resp.TSIGKey

	return
}

func GetServeAlgorithm() {

	libLog.Debug("Entering Akamai Plugin Lib registrar GetServeAlgorithm")

	// both edgedns and this plugin using dns. need to temp swap config...
	existConfig := dns.Config
	defer resetDNSConfig(existConfig)
	dns.Config = *akamaiLibRegistrar.config

	domain := LibPluginArgs.PluginArg.(string)
	zone, err := dns.GetZone(domain)
	if err != nil {
		LibPluginResult.PluginResult = ""
		LibPluginResult.PluginError = err
		return
	}
	libLog.Debugf("Returning Registrar GetServeAlgorithm result")
	LibPluginResult.PluginResult = zone.SignAndServeAlgorithm

	return
}

func GetMasterIPs() {

	libLog.Debug("Entering Akamai Plugin Lib registrar GetMasterIPs")

	// both edgedns and this plugin using dns. need to temp swap config...
	existConfig := dns.Config
	defer resetDNSConfig(existConfig)
	dns.Config = *akamaiLibRegistrar.config

	LibPluginResult.PluginResult = []string{}
	if len(akamaiLibRegistrar.akaConfig.AkamaiContracts) < 1 {
		libLog.Debug("Registrar GetMasterIPs failed. No contracts")
		LibPluginResult.PluginError = fmt.Errorf("No contracts provided")
		return
	}
	contractId := strings.Split(akamaiLibRegistrar.akaConfig.AkamaiContracts, ",")[0]
	masters, err := dns.GetNameServerRecordList(contractId)
	if err != nil {
		libLog.Debugf("Registrar GetMasterIPs failed. Error: %s", err.Error())
		LibPluginResult.PluginError = err
		return
	}

	libLog.Debugf("Akamai Plugin Registrar GetMasterIPs result: %v", masters)
	LibPluginResult.PluginResult = masters

	return
}

func loadConfig(configFile string) (*AkamaiConfig, error) {

	libLog.Debug("Entering Plugin Lib Akamai registrar loadConfig")
	if fileExists(configFile) {
		// Load config from file
		configData, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, err
		}

		return loadConfigContent(configData)
	}

	libLog.Infof("Config file %v does not exist, using default values", configFile)
	return nil, nil

}

func loadConfigContent(configData []byte) (*AkamaiConfig, error) {
	config := AkamaiConfig{}
	err := yaml.Unmarshal(configData, &config)
	if err != nil {
		return nil, err
	}

	libLog.Info("Akamai plugin registrar config loaded")
	libLog.Debugf("Loaded config: %v", config)
	return &config, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func main() {

	fmt.Println("Akamai Plugin Library Registrar")
}
