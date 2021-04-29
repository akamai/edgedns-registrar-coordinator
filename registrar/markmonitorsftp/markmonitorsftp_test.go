// Copyright 2021 MarkMonitor Technologies, Inc.
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

package markmonitorsftp

import (
	"github.com/akamai/edgedns-registrar-coordinator/registrar"
	log "github.com/apex/log"

	"context"
	"fmt"
	"os"

	"github.com/stretchr/testify/assert"
	"testing"
)

var ()

// sftp service Stubb
type StubSftpConfig struct {
	FuncOutput map[string]interface{}
	FuncErrors map[string]string
}

func newStubSftpConfig(ctx context.Context) StubSftpConfig {

	log := ctx.Value("appLog").(*log.Entry)
	log.Info("Initializing StubSftpConfig")

	stubOpenDNSConfig := StubSftpConfig{}
	stubOpenDNSConfig.FuncOutput = map[string]interface{}{}
	stubOpenDNSConfig.FuncErrors = map[string]string{}

	return stubOpenDNSConfig
}

// MarkMonitor Registrar Stub
type StubRegistrar struct {
	registrar.BaseRegistrarProvider
}

func newRegistrarStub(ctx context.Context) StubRegistrar {

	log := ctx.Value("appLog").(*log.Entry)
	log.Info("Initializing RegistrarStub")

	return StubRegistrar{}
}

func initRegistrarStub(ctx context.Context) MarkMonitorSFTPConfig {

	config := MarkMonitorSFTPConfig{
		MarkMonitorSshUser:              "testuser",
		MarkMonitorSshPassword:          "testpw",
		MarkMonitorSshHost:              "testhost",
		MarkMonitorSshPort:              9999,
		MarkMonitorMasterIPs:            []string{"1.2.3.4"},
		MarkMonitorDomainConfigFilePath: "testpath",
	}

	return config
}

func StubCloseSFTPSession(interface{}) {

	return

}

func TestParseZoneData(t *testing.T) {

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar": "Test MarkMonitorSFTP",
		"test case": "TestParseZoneData",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)
	zoneLine := `zone "thinkwhatspossible.es" in { type slave; file "/var/dns-config/dbs/zone.thinkwhatspossible.es.bak"; masters { 64.124.14.39; }; allow-transfer {def_xfer; }; };`
	z := ParseZoneData(appLog, zoneLine)
	assert.Equal(t, z, "thinkwhatspossible.es")
}

func TestParseZoneDataFail(t *testing.T) {

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar": "Test MarkMonitorSFTP",
		"test case": "TestParseZoneDataFail",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)
	zoneLine := `zone "thinkwhatspossible.es" in { type mmmmm; file "/var/dns-config/dbs/zone.thinkwhatspossible.es.bak"; masters { 64.124.14.39; }; allow-transfer {def_xfer; }; };`
	z := ParseZoneData(appLog, zoneLine)
	assert.Equal(t, z, "")
}

func TestRegistrarGetDomains(t *testing.T) {

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar": "Test MarkMonitorSFTP",
		"test case": "GetDomains",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(ctx)
	testRegistrar, err := NewMarkMonitorSFTPRegistrar(ctx, config, newStubSftpConfig(ctx))
	if err != nil {
		fmt.Println("err: ", err.Error())
	}
	testRegistrar.closeSFTPSession = StubCloseSFTPSession
	testRegistrar.sftpService.(StubSftpConfig).FuncOutput["ReadRemoteDomainFile"] = &[]string{"one.com", "two.com"}
	doms, err := testRegistrar.GetDomains(ctx)
	assert.Nil(t, err)
	assert.Equal(t, len(doms), 2)
	assert.Equal(t, doms[0], "one.com")
}

func TestRegistrarGetDomainsFailRead(t *testing.T) {

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar": "Test MarkMonitorSFTP",
		"test case": "GetDomainsFailRead",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(ctx)
	testRegistrar, err := NewMarkMonitorSFTPRegistrar(ctx, config, newStubSftpConfig(ctx))
	if err != nil {
		fmt.Println("err: ", err.Error())
	}
	testRegistrar.closeSFTPSession = StubCloseSFTPSession
	testRegistrar.sftpService.(StubSftpConfig).FuncErrors["ReadRemoteDomainFile"] = "Read domain file failed"
	_, err = testRegistrar.GetDomains(ctx)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed")

}

func TestRegistrarGetDomainsFailConn(t *testing.T) {

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar": "Test MarkMonitorSFTP",
		"test case": "GetDomainsFailConn",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(ctx)
	testRegistrar, err := NewMarkMonitorSFTPRegistrar(ctx, config, newStubSftpConfig(ctx))
	if err != nil {
		fmt.Println("err: ", err.Error())
	}
	testRegistrar.closeSFTPSession = StubCloseSFTPSession
	testRegistrar.sftpService.(StubSftpConfig).FuncErrors["EstablishSFTPSession"] = "EstablishSFTPSession failed"
	_, err = testRegistrar.GetDomains(ctx)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed")

}

func TestRegistrarGetDomain(t *testing.T) {

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar": "Test MarkMonitorSFTP",
		"test case": "GetDomain",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(ctx)
	testRegistrar, err := NewMarkMonitorSFTPRegistrar(ctx, config, newStubSftpConfig(ctx))
	if err != nil {
		fmt.Println("err: ", err.Error())
	}
	testRegistrar.closeSFTPSession = StubCloseSFTPSession
	dom, err := testRegistrar.GetDomain(ctx, "test")
	assert.Nil(t, err)
	assert.Equal(t, dom.Name, "")

}

func TestRegistrarGetTsigKey(t *testing.T) {

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar": "Test MarkMonitorSFTP",
		"test case": "GetTsigKey",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(ctx)
	testRegistrar, err := NewMarkMonitorSFTPRegistrar(ctx, config, newStubSftpConfig(ctx))
	if err != nil {
		fmt.Println("err: ", err.Error())
	}
	testRegistrar.closeSFTPSession = StubCloseSFTPSession
	tkey, err := testRegistrar.GetTsigKey(ctx, "test")
	assert.Nil(t, err)
	assert.Nil(t, tkey, nil)
}

func TestRegistrarGetServeAlgorithm(t *testing.T) {

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar": "Test MarkMonitorSFTP",
		"test case": "GetServeAlgorithm",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(ctx)
	testRegistrar, err := NewMarkMonitorSFTPRegistrar(ctx, config, newStubSftpConfig(ctx))
	if err != nil {
		fmt.Println("err: ", err.Error())
	}
	testRegistrar.closeSFTPSession = StubCloseSFTPSession
	algo, err := testRegistrar.GetServeAlgorithm(ctx, "test")
	assert.Nil(t, err)
	assert.Equal(t, algo, "")
}

func TestRegistrarGetMasterIPs(t *testing.T) {

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar": "Test MarkMonitorSFTP",
		"test case": "GetGetMasterIPs",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(ctx)
	testRegistrar, err := NewMarkMonitorSFTPRegistrar(ctx, config, newStubSftpConfig(ctx))
	if err != nil {
		fmt.Println("err: ", err.Error())
	}
	testRegistrar.closeSFTPSession = StubCloseSFTPSession
	ips, err := testRegistrar.GetMasterIPs(ctx)
	assert.Nil(t, err)
	assert.Equal(t, len(ips), 1)
	assert.Equal(t, ips[0], "1.2.3.4")
}

//
//  Stubbable functions
//

// establish SFTPSession if not already
func (s StubSftpConfig) EstablishSFTPSession(log *log.Entry, markmonitorConfig *MarkMonitorSFTPConfig) error {

	errmsg, ok := s.FuncErrors["EstablishSFTPSession"]
	if ok {
		err := fmt.Errorf(errmsg)
		log.Debugf("error: %s", err.Error())
		return err
	}

	return nil
}

// ParseDomainFile parses reteieved domains file. Returns map of domains indexed by masterp ip and error
func (s StubSftpConfig) ParseDomainFile(log *log.Entry, domFile *os.File) (*[]string, error) {

	return &[]string{}, nil

}

// ReadRemoteDomainFile reads remote dmains file, saves to remp location. returns handle of temp file and error.
func (s StubSftpConfig) ReadRemoteDomainFile(log *log.Entry, markmonitorConfig *MarkMonitorSFTPConfig) (*[]string, error) {

	var err error
	domainsr, ok := s.FuncOutput["ReadRemoteDomainFile"]
	if ok {
		domList := domainsr.(*[]string)
		log.Debugf("ReadRemoteDomainFile: [%v]", domList)
		return domList, nil
	}
	errmsg, ok := s.FuncErrors["ReadRemoteDomainFile"]
	if !ok {
		err = fmt.Errorf("ReadRemoteDomainFile expected output. Got none")
	} else {
		err = fmt.Errorf(errmsg)
	}
	log.Debugf("error: %s", err.Error())
	return nil, err
}
