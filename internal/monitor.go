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
	"github.com/akamai/edgedns-registrar-coordinator/registrar"
	"github.com/apex/log"

	"context"
	"strconv"
	"time"
)

var (
	// track last registrar domain list
	lastRegistrarTally = map[string]map[string]bool{}
)

func Monitor(ctx context.Context, err chan string, regname string, reg registrar.RegistrarProvider, edge *EdgeDNSHandler, interval time.Duration, dryrun bool, once bool) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering Monitor")
	queryArgs := dns.ZoneListQueryArgs{
		ContractIds: edge.Contract,
		ShowAll:     true,
		SortBy:      "zone",
		Types:       "SECONDARY",
	}
	log.Debugf("Edge CONTRACT: %s", edge.Contract)
	for {
		nextLoop := time.Now().Add(interval)
		edgeZones, edgeErr := edge.GetZoneNames(ctx, queryArgs, []string{"LOCKED"})
		registrarDomains, regErr := reg.GetDomains(ctx) // Up to registrar to decide how to filter

		if edgeErr != nil {
			log.Errorf("Monitor. Failed to read EdgeDNS Secondary zones. Error: %s", edgeErr.Error())
		} else if regErr != nil {
			log.Errorf("Monitor. Failed to read registrar primary zones. Error: %s", regErr.Error())
		} else {
			log.Debugf("Monitor. Retrieved Edge DNS zones: %v", edgeZones)
			log.Debugf("Monitor. Retrieved Registrar zones: %v", registrarDomains)
			// process
			newZones, removedZones := diffZoneLists(ctx, regname, edgeZones, registrarDomains)
			err := addSecondaryZones(ctx, edge, reg, newZones, dryrun)
			if err != nil {
				log.Errorf("Monitor. Failed to add secondary zones. Error: %s", err.Error())
			}
			removeSecondaryZones(ctx, edge, removedZones, dryrun)
			if err != nil {
				log.Errorf("Monitor. Failed to remove secondary zones. Error: %s", err.Error())
			}

		}
		if once {
			log.Debug("Monitor executed once. Exiting")
			break
		}
		loopEnd := time.Now()
		if nextLoop.After(loopEnd) {
			time.Sleep(nextLoop.Sub(loopEnd))
		}
	}

	err <- ""

}

func diffZoneLists(ctx context.Context, regname string, edgeZones, registrarDomains []string) (newZones []string, removedZones []string) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Monitor. Diffing Edge DNS and Registrar domain lists")

	var lastTally map[string]bool
	lastTally, ok := lastRegistrarTally[regname]
	if !ok {
		lastTally = make(map[string]bool)
	}
	// Assme the register list is smaller than edgedns list ...
	edgehash := make(map[string]bool)
	reghash := make(map[string]bool)

	for _, e := range edgeZones {
		edgehash[e] = true
	}
	for _, e := range registrarDomains {
		reghash[e] = true
		if _, ok := edgehash[e]; !ok {
			// not there. new
			log.Debugf("New zone to create: %s", e)
			newZones = append(newZones, e)
		}
	}
	for d, _ := range lastTally {
		if _, ok := reghash[d]; !ok {
			if _, ok := edgehash[d]; ok {
				log.Debugf("Zone to delete: %s", d)
				removedZones = append(removedZones, d)
			}
		}
	}
	// Save current for next round
	lastRegistrarTally[regname] = reghash

	return

}

func addSecondaryZones(ctx context.Context, edge *EdgeDNSHandler, reg registrar.RegistrarProvider, newZones []string, dryrun bool) error {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debugf("Monitor. addSecondaryZones: %v", newZones)
	if len(newZones) < 1 {
		return nil
	}
	masters, err := reg.GetMasterIPs(ctx)
	if err != nil {
		log.Warnf("Unable to retrieve master Ips. Error: %s", err.Error())
		return err
	}
	// Create **Seconday** Zones one at a time ...
	for _, zname := range newZones {
		zonequerystring := dns.ZoneQueryString{Contract: edge.Contract, Group: strconv.Itoa(edge.Group)}
		zone := &dns.ZoneCreate{Zone: zname, Type: "Secondary", Comment: "Created by EdgeDNS Registrar Coordinator"}
		zone.Masters = masters
		if edge.DNSSEC {
			if algo, err := reg.GetServeAlgorithm(ctx, zname); err == nil && algo != "" {
				zone.SignAndServe = true
				zone.SignAndServeAlgorithm = algo
			} else {
				log.Warn("Unable to retrieve Sign algorithm")
			}
		}
		if edge.TSig {
			tsigKey, err := reg.GetTsigKey(ctx, zname)
			if tsigKey != nil && err == nil {
				zone.TsigKey = tsigKey // tsig key
			} else {
				log.Warn("Unable to retrieve TSig Key")
			}
		}
		if dryrun {
			log.Infof("Add secondary zone %s. dry run. No changes made", zname)
			log.Debugf("Secondary zone: %v", zone)
			continue
		}
		err := edge.CreateZone(ctx, zone, zonequerystring)
		if err != nil {
			log.Errorf("Create zone error. %s", err.Error())
		}
	}

	return nil

}

func removeSecondaryZones(ctx context.Context, edge *EdgeDNSHandler, removedZones []string, dryrun bool) error {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debugf("removeSecondaryZones: %v", removedZones)
	if len(removedZones) < 1 {
		return nil
	}
	if dryrun {
		log.Infof("Remove secondary zones: [%v]. dry run. No changes made", removedZones)
		return nil
	}

	zonelist := &dns.ZoneNameListResponse{Zones: removedZones}
	_, err := edge.DeleteBulkZones(ctx, zonelist) // (*dns.BulkZonesResponse, error)
	if err != nil {
		log.Errorf("Delete zones error. %s", err.Error())
		return err
	}

	return nil

}
