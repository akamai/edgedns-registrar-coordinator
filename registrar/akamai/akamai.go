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
	log "github.com/apex/log"
	"github.com/akamai/edgedns-registrar-coordinator/registrar"

	dns "github.com/akamai/AkamaiOPEN-edgegrid-golang/configdns-v2"
	edgegrid "github.com/akamai/AkamaiOPEN-edgegrid-golang/edgegrid"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"context"
	"os"
	"strconv"
	"strings"
	"time"
	"fmt"
)

const ()

var ()

// edgeDNSClient is a proxy interface of the Akamai edgegrid configdns-v2 package that can be stubbed for testing.
type AkamaiDNSService interface {
	GetDomains(ctx context.Context) ([]string, error)
	GetDomain(ctx context.Context, domain string) (*registrar.Domain, error)
	GetTsigKey(ctx context.Context, domain string) (*dns.TSIGKey, error)
	GetServeAlgorithm(ctx context.Context, domain string) (string, error)
	GetMasterIPs(ctx context.Context) ([]string, error)
}

// AkamaiProvider implements the DNS provider for Akamai.
type AkamaiRegistrar struct {
	registrar.BaseRegistrarProvider
	akamaiConfig *AkamaiConfig
	config       *edgegrid.Config
	dryRun       bool
	once         bool
	// Defines client. Allows for mocking.
	client AkamaiDNSService
}

type AkamaiConfig struct {
	AkamaiConfigPath    string
	AkamaiContracts     string `yaml:"akamai_contracts"`
	AkamaiNameFilter    string `yaml:"akamai_name_filter"`
	AkamaiEdgercPath    string `yaml:"akamai_edgerc_path"`
	AkamaiEdgercSection string `yaml:"akamai_edgerc_section"`
	AkamaiHost          string `yaml:"akamai_host"`
	AkamaiAccessToken   string `yaml:"akamai_access_token"`
	AkamaiClientToken   string `yaml:"akamai_client_token"`
	AkamaiClientSecret  string `yaml:"akamai_client_secret"`
	Interval            time.Duration
	MaxBody             int
	AccountKey          string
	DryRun              bool
	Once                bool
}

// NewAkamaiProvider initializes a new Akamai DNS based Provider.
func NewAkamaiRegistrar(ctx context.Context, akamaiConfig AkamaiConfig, akaService AkamaiDNSService) (*AkamaiRegistrar, error) {

	var akaConfig *AkamaiConfig
	var err error

	log := ctx.Value("appLog").(*log.Entry)
	// Get file config and parse
	if akamaiConfig.AkamaiConfigPath != "" {
		akaConfig, err = loadConfig(log, akamaiConfig.AkamaiConfigPath)
		if err != nil {
			return nil, err
		}
	}
	if akaConfig != nil {
		// Command line over rides ...
		if len(akamaiConfig.AkamaiContracts) == 0 {
			akamaiConfig.AkamaiContracts = akaConfig.AkamaiContracts
		}
		if akamaiConfig.AkamaiNameFilter == "" {
			akamaiConfig.AkamaiNameFilter = akaConfig.AkamaiNameFilter
		}
		if akamaiConfig.AkamaiEdgercPath == "" {
			akamaiConfig.AkamaiEdgercPath = akaConfig.AkamaiEdgercPath
		}
		if akamaiConfig.AkamaiEdgercSection == "" {
			akamaiConfig.AkamaiEdgercSection = akaConfig.AkamaiEdgercSection
		}
		if akamaiConfig.AkamaiHost == "" {
			akamaiConfig.AkamaiHost = akaConfig.AkamaiHost
		}
		if akamaiConfig.AkamaiAccessToken == "" {
			akamaiConfig.AkamaiAccessToken = akaConfig.AkamaiAccessToken
		}
		if akamaiConfig.AkamaiClientToken == "" {
			akamaiConfig.AkamaiClientToken = akaConfig.AkamaiClientToken
		}
		if akamaiConfig.AkamaiClientSecret == "" {
			akamaiConfig.AkamaiClientSecret = akaConfig.AkamaiClientSecret
		}
	}
	// Process creds. Could be on cmd line, config file
	var edgeGridConfig edgegrid.Config

	log.Debugf("Host: %s", akamaiConfig.AkamaiHost)
	log.Debugf("ClientToken: %s", akamaiConfig.AkamaiClientToken)
	log.Debugf("ClientSecret: %s", akamaiConfig.AkamaiClientSecret)
	log.Debugf("AccessToken: %s", akamaiConfig.AkamaiAccessToken)
	log.Debugf("EdgePath: %s", akamaiConfig.AkamaiEdgercPath)
	log.Debugf("EdgeSection: %s", akamaiConfig.AkamaiEdgercSection)
	log.Debugf("AkamaiContracts: %v", akamaiConfig.AkamaiContracts)
	// environment overrides edgerc file but config needs to be complete
	if akamaiConfig.AkamaiHost == "" || akamaiConfig.AkamaiClientToken == "" || akamaiConfig.AkamaiClientSecret == "" || akamaiConfig.AkamaiAccessToken == "" {
		// Look for Akamai environment or .edgerd creds
		var err error
		edgeGridConfig, err = edgegrid.Init(akamaiConfig.AkamaiEdgercPath, akamaiConfig.AkamaiEdgercSection) // use default .edgerc location and section
		if err != nil {
			log.Errorf("Edgegrid Init Failed")
			return nil, err // return empty provider for backward compatibility
		}
	} else {
		// Use external-dns config
		edgeGridConfig = edgegrid.Config{
			Host:         akamaiConfig.AkamaiHost,
			ClientToken:  akamaiConfig.AkamaiClientToken,
			ClientSecret: akamaiConfig.AkamaiClientSecret,
			AccessToken:  akamaiConfig.AkamaiAccessToken,
			MaxBody:      131072, // same default val as used by Edgegrid
			Debug:        false,
		}
		// Check for edgegrid overrides
		if envval, ok := os.LookupEnv("AKAMAI_MAX_BODY"); ok {
			if i, err := strconv.Atoi(envval); err == nil {
				edgeGridConfig.MaxBody = i
				log.Debugf("Edgegrid maxbody set to %s", envval)
			}
		}
		if envval, ok := os.LookupEnv("AKAMAI_ACCOUNT_KEY"); ok {
			edgeGridConfig.AccountKey = envval
			log.Debugf("Edgegrid applying account key %s", envval)
		}
		if envval, ok := os.LookupEnv("AKAMAI_DEBUG"); ok {
			if dbgval, err := strconv.ParseBool(envval); err == nil {
				edgeGridConfig.Debug = dbgval
				log.Debugf("Edgegrid debug set to %s", envval)
			}
		}
	}

	provider := &AkamaiRegistrar{
		config:       &edgeGridConfig,
		akamaiConfig: &akamaiConfig,
		dryRun:       akamaiConfig.DryRun,
		once:         akamaiConfig.Once,
	}
	if akaService != nil {
		log.Debugf("Using STUB")
		provider.client = akaService
	} else {
		provider.client = provider
	}

	// Init library for direct endpoint calls
	dns.Init(edgeGridConfig)

	return provider, nil
}

func (a *AkamaiRegistrar) GetDomains(ctx context.Context) ([]string, error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering Akamai registrar GetDomains")
	queryArgs := dns.ZoneListQueryArgs{
		Types:       "PRIMARY",
		SortBy:      "zone",
		ContractIds: a.akamaiConfig.AkamaiContracts,
		Search:      a.akamaiConfig.AkamaiNameFilter,
	}
	log.Debugf("ListZones Query Args: %v", queryArgs)
	zlResp, err := dns.ListZones(queryArgs)
	if err != nil {
		log.Debugf("Registrar GetDomains failed. Error: %s", err.Error())
		return []string{}, err
	}
	filter := "LOCKED"
	domains := make([]string, 0, len(zlResp.Zones))
	for _, zone := range zlResp.Zones {
		if strings.Contains(filter, zone.ActivationState) {
			continue
		}
		domains = append(domains, zone.Zone)
	}

	log.Debugf("Registrar GetDomains result: %v", domains)
	return domains, nil
}

func (a *AkamaiRegistrar) GetDomain(ctx context.Context, domain string) (*registrar.Domain, error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering Akamai registrar GetDomain")
	zone, err := dns.GetZone(domain)
	if err != nil {
		return nil, err
	}
	log.Debugf("Registrar GetDomain result: %v", zone)
	return &registrar.Domain{
		Name:                  zone.Zone,
		Type:                  zone.Type,
		SignAndServe:          zone.SignAndServe,
		SignAndServeAlgorithm: zone.SignAndServeAlgorithm,
		Masters:               zone.Masters,
		TsigKey:               zone.TsigKey,
	}, nil
}

func (a *AkamaiRegistrar) GetTsigKey(ctx context.Context, domain string) (tsigKey *dns.TSIGKey, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering Akamai registrar GetTsigKey")
	resp, err := dns.GetZoneKey(domain)

	if err != nil {
		return nil, err
	}

	log.Debugf("Returning Registrar GetTsigKey result")
	return &resp.TSIGKey, nil
}

func (a *AkamaiRegistrar) GetServeAlgorithm(ctx context.Context, domain string) (algo string, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering Akamai registrar GetServeAlgorithm")
	zone, err := dns.GetZone(domain)
	if err != nil {
		return "", nil
	}
	log.Debugf("Returning Registrar GetServeAlgorithm result")
	return zone.SignAndServeAlgorithm, nil
}

func (a *AkamaiRegistrar) GetMasterIPs(ctx context.Context) ([]string, error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering Akamai registrar GetMasterIPs")
	if len(a.akamaiConfig.AkamaiContracts) < 1 {
		log.Debug("Registrar GetMasterIPs failed. No contracts")
		return []string{}, fmt.Errorf("No contracts provided")
	}
	contractId := strings.Split(a.akamaiConfig.AkamaiContracts, ",")[0]
	masters, err := dns.GetNameServerRecordList(contractId)
	if err != nil {
		log.Debugf("Registrar GetMasterIPs failed. Error: %s", err.Error())
		return []string{}, err
	}

	log.Debugf("Registrar GetMasterIPs result: %v", masters)
	return masters, nil
}

func loadConfig(log *log.Entry, configFile string) (*AkamaiConfig, error) {

	log.Debug("Entering Akamai registrar loadConfig")
	if fileExists(configFile) {
		// Load config from file
		configData, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, err
		}

		return loadConfigContent(log, configData)
	}

	log.Infof("Config file %v does not exist, using default values", configFile)
	return nil, nil

}

func loadConfigContent(log *log.Entry, configData []byte) (*AkamaiConfig, error) {
	config := AkamaiConfig{}
	err := yaml.Unmarshal(configData, &config)
	if err != nil {
		return nil, err
	}

	log.Info("akamai registrar config loaded")
	return &config, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
