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
	dns "github.com/akamai/AkamaiOPEN-edgegrid-golang/configdns-v2"
	edgegrid "github.com/akamai/AkamaiOPEN-edgegrid-golang/edgegrid"

	"context"
	"github.com/apex/log"
	"os"
	"strconv"
	"strings"
)

const ()

var (
	edgeDNSHandler *EdgeDNSHandler
)

// edgeDNSClient is a proxy interface of the Akamai edgegrid configdns-v2 package that can be stubbed for testing.
type AkamaiDNSService interface {
	GetZoneNames(ctx context.Context, queryArgs dns.ZoneListQueryArgs, stateFilter []string) ([]string, error)
	GetZones(ctx context.Context, queryArgs dns.ZoneListQueryArgs) (*dns.ZoneListResponse, error)
	GetZone(ctx context.Context, zone string) (*dns.ZoneResponse, error)
	CreateZone(ctx context.Context, zone *dns.ZoneCreate, zonequerystring dns.ZoneQueryString) error
	CreateBulkZones(ctx context.Context, bulkzones *dns.BulkZonesCreate, zonequerystring dns.ZoneQueryString) (*dns.BulkZonesResponse, error)
	DeleteBulkZones(ctx context.Context, zoneslist *dns.ZoneNameListResponse) (*dns.BulkZonesResponse, error)
	//DeleteZone(zone *dns.ZoneCreate, zonequerystring dns.ZoneQueryString) error
}

type EdgeDNSHandler struct {
	Contract      string
	Group         int
	DNSSEC        bool
	TSig          bool
	Host          string
	ClientToken   string
	ClientSecret  string
	AccessToken   string
	EdgercPath    string
	EdgercSection string
	config        *edgegrid.Config
	// Defines client. Allows for mocking.
	client AkamaiDNSService
}

// NewAkamaiProvider initializes a new Akamai DNS based Provider.
func InitEdgeDNSHandler(ctx context.Context, config *Config, akaService AkamaiDNSService) (*EdgeDNSHandler, error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Initializing EdgeDNSHandler")
	edgeDNSHandler = &EdgeDNSHandler{
		Contract:      config.EdgeDNSContract,
		Group:         config.EdgeDNSGroup,
		DNSSEC:        config.DNSSEC,
		TSig:          config.TSig,
		Host:          config.EdgegridHost,
		ClientToken:   config.EdgegridClientToken,
		ClientSecret:  config.EdgegridClientSecret,
		AccessToken:   config.EdgegridAccessToken,
		EdgercPath:    config.EdgegridEdgercPath,
		EdgercSection: config.EdgegridEdgercSection,
		//Logger                  string
		//LogLevel                string
	}

	// Process creds
	var edgeGridConfig edgegrid.Config

	log.Debugf("Host: %s", edgeDNSHandler.Host)
	log.Debugf("ClientToken: %s", edgeDNSHandler.ClientToken)
	log.Debugf("ClientSecret: %s", edgeDNSHandler.ClientSecret)
	log.Debugf("AccessToken: %s", edgeDNSHandler.AccessToken)
	log.Debugf("EdgePath: %s", edgeDNSHandler.EdgercPath)
	log.Debugf("EdgeSection: %s", edgeDNSHandler.EdgercSection)

	// environment overrides edgerc file but config needs to be complete
	if edgeDNSHandler.Host == "" || edgeDNSHandler.ClientToken == "" || edgeDNSHandler.ClientSecret == "" || edgeDNSHandler.AccessToken == "" {
		var err error
		edgeGridConfig, err = edgegrid.Init(edgeDNSHandler.EdgercPath, edgeDNSHandler.EdgercSection) // use default .edgerc location and section
		if err != nil {
			log.Errorf("EdgeDNS Edgegrid Init Failed")
			return nil, err
		}
	} else {
		// Use external-dns config
		edgeGridConfig = edgegrid.Config{
			Host:         edgeDNSHandler.Host,
			ClientToken:  edgeDNSHandler.ClientToken,
			ClientSecret: edgeDNSHandler.ClientSecret,
			AccessToken:  edgeDNSHandler.AccessToken,
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

	edgeDNSHandler.config = &edgeGridConfig

	if akaService != nil {
		log.Debugf("EdgeDNS Handler using STUB")
		edgeDNSHandler.client = akaService
	} else {
		edgeDNSHandler.client = edgeDNSHandler
	}

	// Init library for direct endpoint calls
	dns.Init(edgeGridConfig)

	return edgeDNSHandler, nil
}

//
func (e *EdgeDNSHandler) GetZoneNames(ctx context.Context, queryArgs dns.ZoneListQueryArgs, stateFilter []string) ([]string, error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering EdgeDNS Handler GetZoneNames")
	// type shud be set to SECONDARY!
	zlResp, err := e.GetZones(ctx, queryArgs)
	if err != nil {
		return []string{}, err
	}
	filter := strings.Join(stateFilter, " ")
	zones := make([]string, 0, len(zlResp.Zones))
	for _, zone := range zlResp.Zones {
		if strings.Contains(filter, zone.ActivationState) {
			continue
		}
		zones = append(zones, zone.Zone)
	}

	log.Debugf("GetZoneNames result: %v", zones)
	return zones, nil
}

func (e *EdgeDNSHandler) GetZones(ctx context.Context, queryArgs dns.ZoneListQueryArgs) (*dns.ZoneListResponse, error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering EdgeDNS Handler GetZones")
	log.Debugf("queryArgs: %v", queryArgs)
	zoneresp, err := dns.ListZones(queryArgs)
	if err == nil {
		log.Debugf("GetZones result: %v", zoneresp)
	}
	return zoneresp, err

}

func (e *EdgeDNSHandler) GetZone(ctx context.Context, zone string) (*dns.ZoneResponse, error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering EdgeDNS Handler GetZone")
	zoneresp, err := dns.GetZone(zone)
	if err == nil {
		log.Debugf("GetZone result: %v", zoneresp)
	}
	return zoneresp, err
}

func (e *EdgeDNSHandler) CreateZone(ctx context.Context, zone *dns.ZoneCreate, zonequerystring dns.ZoneQueryString) error {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering EdgeDNS Handler CreateZone")
	log.Debugf("Creating Zone: %v", zone)
	return zone.Save(zonequerystring)

}

/*
// Delete needs to be done thru bulk delete endpoint... and async
func (e *EdgeDNSHandler) DeleteZone(zone *dns.ZoneCreate, zonequerystring dns.ZoneQueryString) error {

	return zone.Delete(zonequerystring)

}
*/

func (e *EdgeDNSHandler) CreateBulkZones(ctx context.Context, bulkzones *dns.BulkZonesCreate, zonequerystring dns.ZoneQueryString) (*dns.BulkZonesResponse, error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering EdgeDNS Handler CreateBulkZones")
	return dns.CreateBulkZones(bulkzones, zonequerystring)
}

func (e *EdgeDNSHandler) DeleteBulkZones(ctx context.Context, zoneslist *dns.ZoneNameListResponse) (*dns.BulkZonesResponse, error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering EdgeDNS Handler DeleteBulkZones")
	return dns.DeleteBulkZones(zoneslist)
}
