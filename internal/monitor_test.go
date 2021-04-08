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
	"fmt"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
	"testing"
)

type EdgednsStub struct {
	FuncOutput map[string]interface{}
	FuncErrors map[string]string
}

func newEdgeDNSStub(ctx context.Context) *EdgednsStub {

	log := ctx.Value("appLog").(*log.Entry)
	log.Info("Initializing EdgeDNSStub")

	return &EdgednsStub{
		FuncOutput: map[string]interface{}{},
		FuncErrors: map[string]string{},
	}
}

func initEdgeDNSStubHandler(ctx context.Context, edgednsStub *EdgednsStub, config Config) *EdgeDNSHandler {

	log := ctx.Value("appLog").(*log.Entry)
	log.Info("Initializing EdgeDNSStubHandler")

	handler, _ := InitEdgeDNSHandler(ctx, &config, edgednsStub)

	return handler
}

type StubRegistrar struct {
	registrar.BaseRegistrarProvider
	FuncOutput map[string]interface{}
	FuncErrors map[string]string
}

func newRegistrarStub(ctx context.Context) StubRegistrar {

	log := ctx.Value("appLog").(*log.Entry)
	log.Info("Initializing RegistrarStub")

	return StubRegistrar{
		FuncOutput: map[string]interface{}{},
		FuncErrors: map[string]string{},
	}
}

func initStubs(ctx context.Context) (StubRegistrar, *EdgednsStub, Config) {

	// Create stub Registrar
	stubRegistrar := newRegistrarStub(ctx)
	// Create stub EdgeDNS
	stubEdgeDNS := newEdgeDNSStub(ctx)

	// Populate registrar test data
	stubRegistrar.FuncOutput["GetDomains"] = []string{"regtest.zone", "regtest2.zone"}
	stubRegistrar.FuncOutput["GetTsigKey"] = &dns.TSIGKey{}
	stubRegistrar.FuncOutput["GetServeAlgorithm"] = "1234567890abcdefghijklmnop"
	stubRegistrar.FuncOutput["GetMasterIPs"] = []string{"1.2.3.4", "5.6.7.8"}
	//stubRegistar.FuncOutput["GetDomain"] = &registrar.Domain{}

	// Populate EdgeDNSHander test data
	stubEdgeDNS.FuncOutput["GetZoneNames"] = []string{"edgetest.zone", "edgetest2.zone"}
	stubEdgeDNS.FuncOutput["DeleteBulkZones"] = &dns.BulkZonesResponse{}
	//stubEdgeDNS.FuncOutput["GetZones"] := &dns.ZoneListResponse{}
	//stubEdgeDNS.FuncOutput["GetZone"] := &dns.ZoneResponse{}
	//stubEdgeDNS.FuncOutput["CreateZone"] :=
	//stubEdgeDNS.FuncOutput["CreateBulkZones"] := &dns.BulkZonesResponse{}

	config := Config{
		EdgeDNSContract:      "123456",
		EdgeDNSGroup:         123456789,
		DNSSEC:               false,
		TSig:                 false,
		EdgegridHost:         "test host",
		EdgegridClientToken:  "test ClientToken",
		EdgegridClientSecret: "test ClientSecret",
		EdgegridAccessToken:  "test AccessToken",
		FailOnError:          false,
	}

	return stubRegistrar, stubEdgeDNS, config
}

// TestMonitorBasic exercises the Monitor function.
// Monitor(ctx context.Context, err chan string, regname string, reg registrar.RegistrarProvider, edge *EdgeDNSStubHandler, interval time.Duration, dryrun bool, once bool)
func TestMonitorBasic(t *testing.T) {

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test",
		"subcommand": "TestMonitor",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)
	appLog.Info("TestMonitorBasic")

	stubRegistrar, stubEdgeDNS, config := initStubs(ctx)
	lastRegistrarTally["test"] = map[string]bool{"testdelete.zone": true}
	log.Debugf("lastRegistrarTally after: %v", lastRegistrarTally)
	stubEdgeDNS.FuncOutput["GetZoneNames"] = append(stubEdgeDNS.FuncOutput["GetZoneNames"].([]string), "testdelete.zone")

	testInterval := 1 * time.Second
	cmderr := make(chan string)
	handler := initEdgeDNSStubHandler(ctx, stubEdgeDNS, config)
	appLog.Info("Calling Monitor. DNNSEC, TSig, Dryrun all false; Once true")
	go Monitor(ctx, cmderr, "test", stubRegistrar, handler, testInterval, false, true)
	errmsg := <-cmderr
	assert.Equal(t, errmsg, "")
}

// TestMonitorBasic2 exercises the Monitor function with a couple of intervals.
func TestMonitorBasic2(t *testing.T) {

	ctx := context.TODO()

	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test",
		"subcommand": "TestMonitor",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)
	appLog.Info("TestMonitorBasic2")

	stubRegistrar, stubEdgeDNS, config := initStubs(ctx)
	testInterval := 1 * time.Second
	cmderr := make(chan string)
	wait := make(chan bool)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	d := time.Now().Add(10 * time.Second)
	dctx, dcancel := context.WithDeadline(context.Background(), d)
	defer dcancel()

	go func() {

		select {
		case <-ctx.Done():
			cmderr <- ""
			break
		}
	}()

	go func() {

		select {
		case <-dctx.Done():
			wait <- true
			break
		}
	}()

	result := ""
	handler := initEdgeDNSStubHandler(ctx, stubEdgeDNS, config)
	appLog.Info("Calling Monitor. DNNSEC, TSig, Dryrun, Once all false")
	go Monitor(ctx, cmderr, "test", stubRegistrar, handler, testInterval, false, false)
	// execute a few intervals and cancel
	select {
	case <-wait:
		result = "deadline"
	case <-cmderr:
		result = "pass"
		cancel()
	}
	assert.Equal(t, result, "pass")
}

// TestMonitorSecure exercises the Monitor function with DNSSEC and TSig enabled
func TestMonitorSecure(t *testing.T) {

	ctx := context.TODO()

	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test",
		"subcommand": "TestMonitor",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)
	appLog.Info("TestMonitorSecure")

	// Create stub Registrar
	stubRegistrar := newRegistrarStub(ctx)
	// Create stub EdgeDNS
	stubEdgeDNS := newEdgeDNSStub(ctx)

	stubRegistrar, stubEdgeDNS, config := initStubs(ctx)
	config.DNSSEC = true
	config.TSig = true

	testInterval := 1 * time.Second
	cmderr := make(chan string)
	wait := make(chan bool)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	go func() {

		select {
		case <-ctx.Done():
			wait <- true
			break
		}
	}()
	result := ""
	handler := initEdgeDNSStubHandler(ctx, stubEdgeDNS, config)
	appLog.Info("Calling Monitor. DNNSEC, TSig, Once true. Dryrun false")
	go Monitor(ctx, cmderr, "test", stubRegistrar, handler, testInterval, false, true)

	select {
	case <-wait:
		result = "timeout"
	case <-cmderr:
		result = "pass"
		cancel()
	}
	assert.Equal(t, result, "pass")
}

func TestMonitorGetDomainsFail(t *testing.T) {

	ctx := context.TODO()

	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test",
		"subcommand": "TestMonitor",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)
	appLog.Info("TestMonitorDomainsFail")

	stubRegistrar, stubEdgeDNS, config := initStubs(ctx)
	delete(stubRegistrar.FuncOutput, "GetDomains")
	stubRegistrar.FuncErrors["GetDomains"] = "GET failed"
	config.FailOnError = true

	testInterval := 1 * time.Second
	cmderr := make(chan string)
	handler := initEdgeDNSStubHandler(ctx, stubEdgeDNS, config)
	appLog.Info("Calling Monitor. DNNSEC, TSig, Once true. Dryrun false")
	go Monitor(ctx, cmderr, "test", stubRegistrar, handler, testInterval, false, true)
	result := <-cmderr
	log.Debugf("Result: %s", result)
	assert.Equal(t, strings.Contains(result, "Failed"), true)
}

func TestMonitorGetTSigKey(t *testing.T) {

	ctx := context.TODO()

	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test",
		"subcommand": "TestMonitor",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)
	appLog.Info("TestMonitorGetTSigKeyFail")

	stubRegistrar, stubEdgeDNS, config := initStubs(ctx)
	stubRegistrar.FuncOutput["GetTsigKey"].(*dns.TSIGKey).Name = "TestTsigKey"
	stubRegistrar.FuncOutput["GetTsigKey"].(*dns.TSIGKey).Algorithm = "TestAlgorithm"
	stubRegistrar.FuncOutput["GetTsigKey"].(*dns.TSIGKey).Secret = "TestSecret"
	config.FailOnError = true
	config.TSig = true

	testInterval := 1 * time.Second
	cmderr := make(chan string)
	handler := initEdgeDNSStubHandler(ctx, stubEdgeDNS, config)
	appLog.Info("Calling Monitor. DNNSEC, Once, Dryrun false")
	go Monitor(ctx, cmderr, "test", stubRegistrar, handler, testInterval, false, true)

	result := <-cmderr
	log.Debugf("Result: %s", result)
	assert.Equal(t, strings.Contains(result, "Failed"), false)
}

func TestMonitorGetMasterIPsFail(t *testing.T) {

	ctx := context.TODO()

	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test",
		"subcommand": "TestMonitor",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)
	appLog.Info("TestMonitorGetMasterIPsFail")

	stubRegistrar, stubEdgeDNS, config := initStubs(ctx)
	delete(stubRegistrar.FuncOutput, "GetMasterIPs")
	stubRegistrar.FuncErrors["GetMasterIPs"] = "GET failed"
	config.FailOnError = true

	testInterval := 1 * time.Second
	cmderr := make(chan string)
	handler := initEdgeDNSStubHandler(ctx, stubEdgeDNS, config)
	appLog.Info("Calling Monitor. Once true. DNNSEC, TSig, Dryrun false")
	go Monitor(ctx, cmderr, "test", stubRegistrar, handler, testInterval, false, true)

	result := <-cmderr
	log.Debugf("Result: %s", result)
	assert.Equal(t, strings.Contains(result, "Failed"), true)
}

func TestMonitorGetZoneNamesFail(t *testing.T) {

	ctx := context.TODO()

	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test",
		"subcommand": "TestMonitor",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)
	appLog.Info("TestMonitorGetZoneNamesFail")

	stubRegistrar, stubEdgeDNS, config := initStubs(ctx)
	delete(stubEdgeDNS.FuncOutput, "GetZoneNames")
	stubEdgeDNS.FuncErrors["GetZoneNames"] = "GET failed"
	config.FailOnError = true

	testInterval := 1 * time.Second
	cmderr := make(chan string)
	handler := initEdgeDNSStubHandler(ctx, stubEdgeDNS, config)
	appLog.Info("Calling Monitor. DNNSEC, TSig, Once true. Dryrun false")
	go Monitor(ctx, cmderr, "test", stubRegistrar, handler, testInterval, false, true)

	result := <-cmderr
	log.Debugf("Result: %s", result)
	assert.Equal(t, strings.Contains(result, "Failed"), true)
}

func TestMonitorZoneCreateFail(t *testing.T) {

	ctx := context.TODO()

	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test",
		"subcommand": "TestMonitor",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)
	appLog.Info("TestMonitorZoneCreateFail")

	stubRegistrar, stubEdgeDNS, config := initStubs(ctx)
	stubEdgeDNS.FuncErrors["CreateZone"] = "Create failed"
	config.FailOnError = true

	testInterval := 1 * time.Second
	cmderr := make(chan string)
	handler := initEdgeDNSStubHandler(ctx, stubEdgeDNS, config)
	appLog.Info("Calling Monitor. DNNSEC, TSig, Once true. Dryrun false")
	go Monitor(ctx, cmderr, "test", stubRegistrar, handler, testInterval, false, true)

	result := <-cmderr
	log.Debugf("Result: %s", result)
	assert.Equal(t, strings.Contains(result, "Failed"), true)
}

func TestMonitorDeleteBulkFail(t *testing.T) {

	ctx := context.TODO()

	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test",
		"subcommand": "TestMonitor",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)
	appLog.Info("TestMonitorDeleteBulkFail")

	stubRegistrar, stubEdgeDNS, config := initStubs(ctx)
	log.Debugf("lastRegistrarTally before: %v", lastRegistrarTally)
	lastRegistrarTally["test"] = map[string]bool{"testdelete.zone": true}
	log.Debugf("lastRegistrarTally after: %v", lastRegistrarTally)
	stubEdgeDNS.FuncOutput["GetZoneNames"] = append(stubEdgeDNS.FuncOutput["GetZoneNames"].([]string), "testdelete.zone")
	delete(stubEdgeDNS.FuncOutput, "DeleteBulkZones")
	stubEdgeDNS.FuncErrors["DeleteBulkZones"] = "Delete failed"
	config.FailOnError = true

	testInterval := 1 * time.Second
	cmderr := make(chan string)
	handler := initEdgeDNSStubHandler(ctx, stubEdgeDNS, config)
	appLog.Info("Calling Monitor. DNNSEC, TSig, Once true. Dryrun false")
	go Monitor(ctx, cmderr, "test", stubRegistrar, handler, testInterval, false, true)

	result := <-cmderr
	log.Debugf("Result: %s", result)
	assert.Equal(t, strings.Contains(result, "Failed"), true)
}

// stub functions

func (sr StubRegistrar) GetDomains(ctx context.Context) (domains []string, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering STUB Registar GetDomains")

	// Should be return value, either MT or non MT list
	domainsr, ok := sr.FuncOutput["GetDomains"]
	if ok {
		domains = domainsr.([]string)
		log.Debugf("domains: [%v]", domains)
	} else {
		errmsg, ok := sr.FuncErrors["GetDomains"]
		if !ok {
			err = fmt.Errorf("GetDomains expected output. Got none")
		} else {
			err = fmt.Errorf(errmsg)
		}
		log.Debugf("error: %s", err.Error())
	}

	return
}

func (sr StubRegistrar) GetDomain(ctx context.Context, dom string) (domain *registrar.Domain, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering STUB Registar GetDomain")

	domainr, ok := sr.FuncOutput["GetDomain"]
	if ok {
		domain = domainr.(*registrar.Domain)
	} else {
		errmsg, ok := sr.FuncErrors["GetDomain"]
		if !ok {
			err = fmt.Errorf("GetDomain expected output. Got none")
		} else {
			err = fmt.Errorf(errmsg)
		}
	}

	return
}

func (sr StubRegistrar) GetTsigKey(ctx context.Context, domain string) (tsigKey *dns.TSIGKey, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering STUB Registar GetTsigKey")

	tsigKeyr, ok := sr.FuncOutput["GetTsigKey"]
	if ok {
		tsigKey = tsigKeyr.(*dns.TSIGKey)
	} else {
		errmsg, ok := sr.FuncErrors["GetTsigKey"]
		if !ok {
			err = fmt.Errorf("GetTsigKey expected output. Got none")
		} else {
			err = fmt.Errorf(errmsg)
		}
	}

	return
}

func (sr StubRegistrar) GetServeAlgorithm(ctx context.Context, domain string) (algo string, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering STUB Registar GetServeAlgorithm")

	algor, ok := sr.FuncOutput["GetServeAlgorithm"]
	if ok {
		algo = algor.(string)
	} else {
		errmsg, ok := sr.FuncErrors["GetServeAlgorithm"]
		if !ok {
			err = fmt.Errorf("GetServeAlgorithm expected output. Got none")
		} else {
			err = fmt.Errorf(errmsg)
		}
	}

	return
}

func (sr StubRegistrar) GetMasterIPs(ctx context.Context) (masterIps []string, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering STUB Registar GetMasterIPs")

	masterIpsr, ok := sr.FuncOutput["GetMasterIPs"]
	if ok {
		masterIps = masterIpsr.([]string)
	} else {
		errmsg, ok := sr.FuncErrors["GetMasterIPs"]
		if !ok {
			err = fmt.Errorf("GetMasterIPs expected output. Got none")
		} else {
			err = fmt.Errorf(errmsg)
		}
	}

	return
}

func (es *EdgednsStub) GetZoneNames(ctx context.Context, queryArgs dns.ZoneListQueryArgs, stateFilter []string) (zones []string, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering STUB EdgeDNS GetZoneNames")

	zonesr, ok := es.FuncOutput["GetZoneNames"]
	if ok {
		zones = zonesr.([]string)
	} else {
		errmsg, ok := es.FuncErrors["GetZoneNames"]
		if !ok {
			err = fmt.Errorf("GetZoneNames expected output. Got none")
		} else {
			err = fmt.Errorf(errmsg)
		}
	}

	return
}

func (es *EdgednsStub) GetZones(ctx context.Context, queryArgs dns.ZoneListQueryArgs) (zlr *dns.ZoneListResponse, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering STUB EdgeDNS GetZones")

	zlrr, ok := es.FuncOutput["GetZones"]
	if ok {
		zlr = zlrr.(*dns.ZoneListResponse)
	} else {
		errmsg, ok := es.FuncErrors["GetZones"]
		if !ok {
			err = fmt.Errorf("GetZones expected output. Got none")
		} else {
			err = fmt.Errorf(errmsg)
		}
	}

	return
}

func (es *EdgednsStub) GetZone(ctx context.Context, zone string) (zr *dns.ZoneResponse, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering STUB EdgeDNS GetZone")

	zrr, ok := es.FuncOutput["GetZone"]
	if ok {
		zr = zrr.(*dns.ZoneResponse)
	} else {
		errmsg, ok := es.FuncErrors["GetZone"]
		if !ok {
			err = fmt.Errorf("GetZone expected output. Got none")
		} else {
			err = fmt.Errorf(errmsg)
		}
	}

	return
}

func (es *EdgednsStub) CreateZone(ctx context.Context, zone *dns.ZoneCreate, zonequerystring dns.ZoneQueryString) error {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering STUB EdgeDNS CreateZone")

	var err error
	if _, ok := es.FuncOutput["CreateZone"]; ok {
		err = fmt.Errorf("CreateZone should NOT have any output")
		return err
	}
	if errmsg, ok := es.FuncErrors["CreateZone"]; ok {
		err = fmt.Errorf(errmsg)
		fmt.Println("CreateZone Error: ", errmsg)
	}

	return err
}

func (es *EdgednsStub) CreateBulkZones(ctx context.Context, bulkzones *dns.BulkZonesCreate, zonequerystring dns.ZoneQueryString) (bzr *dns.BulkZonesResponse, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering STUB EdgeDNS CreateBulkZones")

	bzrr, ok := es.FuncOutput["CreateBulkZones"]
	if ok {
		bzr = bzrr.(*dns.BulkZonesResponse)
	} else {
		errmsg, ok := es.FuncErrors["CreateBulkZones"]
		if !ok {
			err = fmt.Errorf("CreateBulkZones expected output. Got none")
		} else {
			err = fmt.Errorf(errmsg)
		}
	}

	return
}

func (es *EdgednsStub) DeleteBulkZones(ctx context.Context, zoneslist *dns.ZoneNameListResponse) (bzr *dns.BulkZonesResponse, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering STUB EdgeDNS DeleteBulkZones")

	bzrr, ok := es.FuncOutput["DeleteBulkZones"]
	if ok {
		bzr = bzrr.(*dns.BulkZonesResponse) // check make sure type is right?
	} else {
		errmsg, ok := es.FuncErrors["DeleteBulkZones"]
		if !ok {
			err = fmt.Errorf("DeleteBulkZones expected output. Got none")
		} else {
			err = fmt.Errorf(errmsg)
		}
		fmt.Println("DeleteBulkZones Error: ", errmsg)
	}

	return
}
