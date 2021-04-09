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

package akamai

import (
	"github.com/akamai/edgedns-registrar-coordinator/registrar"
	log "github.com/apex/log"

	"context"
	"fmt"

	dns "github.com/akamai/AkamaiOPEN-edgegrid-golang/configdns-v2"

	"github.com/stretchr/testify/assert"
	"testing"
)

const ()

var ()

// OpenEdggrid DNS Stub
type StubOpenDNSConfig struct {
	FuncOutput map[string]interface{}
        FuncErrors map[string]string
}

func newStubOpenDNSConfig(ctx context.Context) StubOpenDNSConfig {

        log := ctx.Value("appLog").(*log.Entry)
        log.Info("Initializing StubOpenDNSConfig")

        stubOpenDNSConfig := StubOpenDNSConfig{}
	stubOpenDNSConfig.FuncOutput =  map[string]interface{}{}
	stubOpenDNSConfig.FuncErrors = map[string]string{}

	zresp := &dns.ZoneListResponse{}
	zresp.Zones = []*dns.ZoneResponse{}
	zresp.Zones = append(zresp.Zones, &dns.ZoneResponse{Zone: "test_zone_1.com"})
        stubOpenDNSConfig.FuncOutput["ListZones"] = zresp
	stubOpenDNSConfig.FuncOutput["GetZone"] = &dns.ZoneResponse{Zone: "test_zone_1.com", SignAndServeAlgorithm: "abcdefg"}
	tsigresp := &dns.TSIGKeyResponse{}
	tsigresp.Name = "tsig"
	tsigresp.Algorithm = "abcd"
	tsigresp.Secret = "boo"
	stubOpenDNSConfig.FuncOutput["GetZoneKey"] =  tsigresp
	stubOpenDNSConfig.FuncOutput["GetNameServerRecordList"] = []string{"1.2.3.4", "5.6.7.8"}

	return stubOpenDNSConfig 
}

// Akamai Registrar Stub
type StubRegistrar struct {
	registrar.BaseRegistrarProvider
}

func newRegistrarStub(ctx context.Context) StubRegistrar {

	log := ctx.Value("appLog").(*log.Entry)
	log.Info("Initializing RegistrarStub")

	return StubRegistrar{}
}

func initRegistrarStub(ctx context.Context) (StubRegistrar, AkamaiConfig) {

	// Create stub Registrar
	stubRegistrar := newRegistrarStub(ctx)

	config := AkamaiConfig{
		AkamaiContracts:	"abcdefg",
		AkamaiHost:         	"test host",
		AkamaiClientToken:  	"test ClientToken",
		AkamaiClientSecret: 	"test ClientSecret",
		AkamaiAccessToken:  	"test AccessToken",
	}

	return stubRegistrar, config
}

func TestRegistrarGetDomains(t *testing.T) {

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test Akamai",
		"subcommand": "GetDomains",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	stubRegistrar, config := initRegistrarStub(ctx)
	testRegistrar, err := NewAkamaiRegistrar(ctx, config, stubRegistrar)
	if err != nil {
		fmt.Println("err: ", err.Error())
	}
	testRegistrar.dnsclient = newStubOpenDNSConfig(ctx) 
	doms, err := testRegistrar.GetDomains(ctx)
	assert.Nil(t, err)
	assert.Equal(t, len(doms), 1)
}

func TestRegistrarGetDomainsFail(t *testing.T) {

        ctx := context.TODO()
        logLevel, _ := log.ParseLevel("info")
        log.SetLevel(logLevel)

        appLog := log.WithFields(log.Fields{
                "registrar":  "Test Akamai",
                "subcommand": "GetDomains",
        })
        ctx = context.WithValue(ctx, "appLog", appLog)

        stubRegistrar, config := initRegistrarStub(ctx)
        testRegistrar, err := NewAkamaiRegistrar(ctx, config, stubRegistrar)
        if err != nil {
                fmt.Println("err: ", err.Error())
        }
        testRegistrar.dnsclient = newStubOpenDNSConfig(ctx)
	delete(testRegistrar.dnsclient.(StubOpenDNSConfig).FuncOutput, "ListZones")
	testRegistrar.dnsclient.(StubOpenDNSConfig).FuncErrors["ListZones"] = "Fail" 

        _, err = testRegistrar.GetDomains(ctx)
        assert.NotNil(t, err)
        assert.Contains(t, err.Error(), "Fail")
}

func TestRegistrarGetDomain(t *testing.T) {

        ctx := context.TODO()
        logLevel, _ := log.ParseLevel("info")
        log.SetLevel(logLevel)

        appLog := log.WithFields(log.Fields{
                "registrar":  "Test Akamai",
                "subcommand": "GetDomain",
        })
        ctx = context.WithValue(ctx, "appLog", appLog)

        stubRegistrar, config := initRegistrarStub(ctx)
        testRegistrar, err := NewAkamaiRegistrar(ctx, config, stubRegistrar)
        if err != nil {
                fmt.Println("err: ", err.Error())
        }
        testRegistrar.dnsclient = newStubOpenDNSConfig(ctx)
        dom, err := testRegistrar.GetDomain(ctx, "test_zone_1.com")
        assert.Equal(t, err, nil)
        assert.Equal(t, dom.Name, "test_zone_1.com")
}

func TestRegistrarGetDomainFail(t *testing.T) {

        ctx := context.TODO()
        logLevel, _ := log.ParseLevel("info")
        log.SetLevel(logLevel)

        appLog := log.WithFields(log.Fields{
                "registrar":  "Test Akamai",
                "subcommand": "GetDomainFail",
        })
        ctx = context.WithValue(ctx, "appLog", appLog)

        stubRegistrar, config := initRegistrarStub(ctx)
        testRegistrar, err := NewAkamaiRegistrar(ctx, config, stubRegistrar)
        if err != nil {
                fmt.Println("err: ", err.Error())
        }
        testRegistrar.dnsclient = newStubOpenDNSConfig(ctx)
        delete(testRegistrar.dnsclient.(StubOpenDNSConfig).FuncOutput, "GetZone")
        testRegistrar.dnsclient.(StubOpenDNSConfig).FuncErrors["GetZone"] = "Fail"
        _, err = testRegistrar.GetDomain(ctx, "test_zone_1.com")
        assert.NotNil(t, err)
        assert.Contains(t, err.Error(), "Fail")
}

func TestRegistrarGetTsigKey(t *testing.T) {

        ctx := context.TODO()
        logLevel, _ := log.ParseLevel("info")
        log.SetLevel(logLevel)

        appLog := log.WithFields(log.Fields{
                "registrar":  "Test Akamai",
                "subcommand": "GetTsigKey",
        })
        ctx = context.WithValue(ctx, "appLog", appLog)

        stubRegistrar, config := initRegistrarStub(ctx)
        testRegistrar, err := NewAkamaiRegistrar(ctx, config, stubRegistrar)
        if err != nil {
                fmt.Println("err: ", err.Error())
        }
        testRegistrar.dnsclient = newStubOpenDNSConfig(ctx)
        key, err := testRegistrar.GetTsigKey(ctx, "test_zone_1.com")
        assert.Equal(t, err, nil)
        assert.Equal(t, key.Name, "tsig")
}

func TestRegistrarGetTsigKeyFail(t *testing.T) {

        ctx := context.TODO()
        logLevel, _ := log.ParseLevel("info")
        log.SetLevel(logLevel)

        appLog := log.WithFields(log.Fields{
                "registrar":  "Test Akamai",
                "subcommand": "GetTsigKeyFail",
        })
        ctx = context.WithValue(ctx, "appLog", appLog)

        stubRegistrar, config := initRegistrarStub(ctx)
        testRegistrar, err := NewAkamaiRegistrar(ctx, config, stubRegistrar)
        if err != nil {
                fmt.Println("err: ", err.Error())
        }
        testRegistrar.dnsclient = newStubOpenDNSConfig(ctx)
        delete(testRegistrar.dnsclient.(StubOpenDNSConfig).FuncOutput, "GetZoneKey")
        testRegistrar.dnsclient.(StubOpenDNSConfig).FuncErrors["GetZoneKey"] = "Fail"
        _, err = testRegistrar.GetTsigKey(ctx, "test_zone_1.com")
        assert.NotNil(t, err)
        assert.Contains(t, err.Error(), "Fail")
}

func TestRegistrarGetServeAlgorithm(t *testing.T) {

        ctx := context.TODO()
        logLevel, _ := log.ParseLevel("info")
        log.SetLevel(logLevel)

        appLog := log.WithFields(log.Fields{
                "registrar":  "Test Akamai",
                "subcommand": "GetServeAlgorithm",
        })
        ctx = context.WithValue(ctx, "appLog", appLog)

        stubRegistrar, config := initRegistrarStub(ctx)
        testRegistrar, err := NewAkamaiRegistrar(ctx, config, stubRegistrar)
        if err != nil {
                fmt.Println("err: ", err.Error())
        }
        testRegistrar.dnsclient = newStubOpenDNSConfig(ctx)
	algo, err := testRegistrar.GetServeAlgorithm(ctx, "test_zone_1.com")
        assert.Equal(t, err, nil)
        assert.Equal(t, algo, "abcdefg")
}

func TestRegistrarGetServeAlgorithmiFail(t *testing.T) {

        ctx := context.TODO()
        logLevel, _ := log.ParseLevel("info")
        log.SetLevel(logLevel)

        appLog := log.WithFields(log.Fields{
                "registrar":  "Test Akamai",
                "subcommand": "GetServeAlgorithmiFail",
        })
        ctx = context.WithValue(ctx, "appLog", appLog)

        stubRegistrar, config := initRegistrarStub(ctx)
        testRegistrar, err := NewAkamaiRegistrar(ctx, config, stubRegistrar)
        if err != nil {
                fmt.Println("err: ", err.Error())
        }
        testRegistrar.dnsclient = newStubOpenDNSConfig(ctx)
        delete(testRegistrar.dnsclient.(StubOpenDNSConfig).FuncOutput, "GetZone")
        testRegistrar.dnsclient.(StubOpenDNSConfig).FuncErrors["GetZone"] = "Fail"
        _, err = testRegistrar.GetServeAlgorithm(ctx, "test_zone_1.com")
        assert.NotNil(t, err)
        assert.Contains(t, err.Error(), "Fail")
}

func TestRegistrarGetMasterIPs(t *testing.T) {

        ctx := context.TODO()
        logLevel, _ := log.ParseLevel("info")
        log.SetLevel(logLevel)

        appLog := log.WithFields(log.Fields{
                "registrar":  "Test Akamai",
                "subcommand": "GetMasterIPs",
        })
        ctx = context.WithValue(ctx, "appLog", appLog)

        stubRegistrar, config := initRegistrarStub(ctx)
        testRegistrar, err := NewAkamaiRegistrar(ctx, config, stubRegistrar)
        if err != nil {
                fmt.Println("err: ", err.Error())
        }
        testRegistrar.dnsclient = newStubOpenDNSConfig(ctx)
	mips, err := testRegistrar.GetMasterIPs(ctx)
        assert.Equal(t, err, nil)
        assert.Equal(t, len(mips), 2)
}

func TestRegistrarGetMasterIPsFail(t *testing.T) {

        ctx := context.TODO()
        logLevel, _ := log.ParseLevel("info")
        log.SetLevel(logLevel)

        appLog := log.WithFields(log.Fields{
                "registrar":  "Test Akamai",
                "subcommand": "GetMasterIPsFail",
        })
        ctx = context.WithValue(ctx, "appLog", appLog)

        stubRegistrar, config := initRegistrarStub(ctx)
        testRegistrar, err := NewAkamaiRegistrar(ctx, config, stubRegistrar)
        if err != nil {
                fmt.Println("err: ", err.Error())
        }
        testRegistrar.dnsclient = newStubOpenDNSConfig(ctx)
        delete(testRegistrar.dnsclient.(StubOpenDNSConfig).FuncOutput, "GetNameServerRecordList")
        testRegistrar.dnsclient.(StubOpenDNSConfig).FuncErrors["GetNameServerRecordList"] = "Fail"
        _, err = testRegistrar.GetMasterIPs(ctx)
        assert.NotNil(t, err)
        assert.Contains(t, err.Error(), "Fail")
}

//
// Open DNS stubbable functions
//
func (o StubOpenDNSConfig) ListZones(queryArgs dns.ZoneListQueryArgs) (*dns.ZoneListResponse, error) {

	var err error
	domainsr, ok := o.FuncOutput["ListZones"]
	if ok {
		zoneresp := domainsr.(*dns.ZoneListResponse)
		log.Debugf("Zone Resp: [%v]", zoneresp)
		return zoneresp, nil
	}
	errmsg, ok := o.FuncErrors["ListZones"]
	if !ok {
		err = fmt.Errorf("ListZones expected output. Got none")
	} else {
		err = fmt.Errorf(errmsg)
	}
	log.Debugf("error: %s", err.Error())
	return nil, err
}

func (o StubOpenDNSConfig) GetZone(domain string) (*dns.ZoneResponse, error) {

        var err error
        domainrr, ok := o.FuncOutput["GetZone"]
        if ok {
                zoneresp := domainrr.(*dns.ZoneResponse)
                log.Debugf("Zone Resp: [%v]", zoneresp)
                return zoneresp, nil
        }
        errmsg, ok := o.FuncErrors["GetZone"]
        if !ok {
                err = fmt.Errorf("GetZone expected output. Got none")
        } else {
                err = fmt.Errorf(errmsg)
        }
        log.Debugf("error: %s", err.Error())
        return nil, err
}

func (o StubOpenDNSConfig) GetZoneKey(domain string) (*dns.TSIGKeyResponse, error) {

        var err error
        tsigrespr, ok := o.FuncOutput["GetZoneKey"]
        if ok {
                tsigresp := tsigrespr.(*dns.TSIGKeyResponse)
                log.Debugf("Tsig Resp: [%v]", tsigresp)
                return tsigresp, nil
        }
        errmsg, ok := o.FuncErrors["GetZoneKey"]
        if !ok {
                err = fmt.Errorf("GetZoneKey expected output. Got none")
        } else {
                err = fmt.Errorf(errmsg)
        }
        log.Debugf("error: %s", err.Error())
        return nil, err
}

func (o StubOpenDNSConfig) GetNameServerRecordList(contractId string) ([]string, error) {

        var err error
        miplistr, ok := o.FuncOutput["GetNameServerRecordList"]
        if ok {
                miplist := miplistr.([]string)
                log.Debugf("Master IPs: [%v]", miplist)
                return miplist, nil
        }
        errmsg, ok := o.FuncErrors["GetNameServerRecordList"]
        if !ok {
                err = fmt.Errorf("GetNameServerRecordList expected output. Got none")
        } else {
                err = fmt.Errorf(errmsg)
        }
        log.Debugf("error: %s", err.Error())
        return []string{}, err
}

