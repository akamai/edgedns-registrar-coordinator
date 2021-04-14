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

package plugin

import (
	dns "github.com/akamai/AkamaiOPEN-edgegrid-golang/configdns-v2"
	"github.com/akamai/edgedns-registrar-coordinator/registrar"
	log "github.com/apex/log"

	"context"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

const ()

var (
	pluginTestMutex = &sync.Mutex{}
)

func initRegistrarStub(entry *log.Entry) registrar.PluginConfig {

	config := registrar.PluginConfig{
		PluginLibPath: "./test-plugin/test-plugin-lib",
		//PluginName       string
		PluginConfigPath: "./test-plugin/test-plugin.yaml",
		LogEntry:         entry,
		//Registrar        *plugin.Plugin
	}

	return config
}

func testSetup(testRegistrar *PluginRegistrar) {

	testRegistrar.pluginTest = true
	testRegistrar.pluginResult.PluginError = nil
	testRegistrar.pluginResult.PluginResult = nil

	return
}

//
// Test Functions
//
func TestRegistrarGetDomainsDirect(t *testing.T) {

	pluginTestMutex.Lock()
	defer pluginTestMutex.Unlock()

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test Plugin",
		"subcommand": "GetDomains",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(appLog)
	testRegistrar, err := NewPluginRegistrar(ctx, config)
	assert.Nil(t, err)

	// Test plugin will take are and place in Result
	testRegistrar.pluginArgs.PluginArg = map[string]interface{}{"FuncOutput": []string{"test1.com", "test2.com"}}

	appLog.Debugf("Invoking %s library GetDomains", testRegistrar.pluginConfig.PluginName)
	testRegistrar.pluginGetDomains()
	assert.Nil(t, testRegistrar.pluginResult.PluginError)
	dl, _ := testRegistrar.pluginResult.PluginResult.([]string)
	assert.Equal(t, dl, []string{"test1.com", "test2.com"})

}

func TestRegistrarGetDomains(t *testing.T) {

	pluginTestMutex.Lock()
	defer pluginTestMutex.Unlock()

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test Plugin",
		"subcommand": "GetDomains",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(appLog)
	testRegistrar, err := NewPluginRegistrar(ctx, config)
	assert.Nil(t, err)

	// Test plugin will take are and place in Result
	testRegistrar.pluginArgs.PluginArg = map[string]interface{}{"FuncOutput": []string{"test1.com", "test2.com"}}
	testSetup(testRegistrar)

	appLog.Debugf("Invoking %s library GetDomains", testRegistrar.pluginConfig.PluginName)
	testRegistrar.GetDomains(ctx)
	assert.Nil(t, testRegistrar.pluginResult.PluginError)
	dl, _ := testRegistrar.pluginResult.PluginResult.([]string)
	assert.Equal(t, dl, []string{"test1.com", "test2.com"})

}

func TestRegistrarGetDomainsFail(t *testing.T) {

	pluginTestMutex.Lock()
	defer pluginTestMutex.Unlock()

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test Plugin",
		"subcommand": "GetDomainsFail",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(appLog)
	testRegistrar, err := NewPluginRegistrar(ctx, config)
	assert.Nil(t, err)

	// Test plugin will take are and place in Result
	testRegistrar.pluginArgs.PluginArg = map[string]interface{}{"FuncErrors": "GetDomainsFail"}
	testSetup(testRegistrar)
	appLog.Debugf("Invoking %s library GetDomains", testRegistrar.pluginConfig.PluginName)
	testRegistrar.GetDomains(ctx)
	assert.NotNil(t, testRegistrar.pluginResult.PluginError)
	assert.Contains(t, testRegistrar.pluginResult.PluginError.Error(), "Fail")

}

func TestRegistrarGetDomain(t *testing.T) {

	pluginTestMutex.Lock()
	defer pluginTestMutex.Unlock()

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test Plugin",
		"subcommand": "GetDomain",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(appLog)
	testRegistrar, err := NewPluginRegistrar(ctx, config)
	assert.Nil(t, err)

	// Test plugin will take are and place in Result
	testRegistrar.pluginArgs.PluginArg = map[string]interface{}{"FuncOutput": &registrar.Domain{Name: "test1.com", Type: "PRIMARY", SignAndServeAlgorithm: "abcdefg"}}
	testSetup(testRegistrar)
	appLog.Debugf("Invoking %s library GetDomain", testRegistrar.pluginConfig.PluginName)
	testRegistrar.GetDomain(ctx, "test1.com")
	assert.Nil(t, testRegistrar.pluginResult.PluginError)
	dl, _ := testRegistrar.pluginResult.PluginResult.(*registrar.Domain)
	assert.Equal(t, dl.Name, "test1.com")

}

func TestRegistrarGetDomainFail(t *testing.T) {

	pluginTestMutex.Lock()
	defer pluginTestMutex.Unlock()

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test Plugin",
		"subcommand": "GetDomainFail",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(appLog)
	testRegistrar, err := NewPluginRegistrar(ctx, config)
	assert.Nil(t, err)

	// Test plugin will take are and place in Result
	testRegistrar.pluginArgs.PluginArg = map[string]interface{}{"FuncErrors": "GetDomainFail"}
	testSetup(testRegistrar)
	appLog.Debugf("Invoking %s library GetDomain", testRegistrar.pluginConfig.PluginName)
	testRegistrar.GetDomain(ctx, "test1.com")
	assert.NotNil(t, testRegistrar.pluginResult.PluginError)
	assert.Contains(t, testRegistrar.pluginResult.PluginError.Error(), "Fail")

}

func TestRegistrarGetTsigKey(t *testing.T) {

	pluginTestMutex.Lock()
	defer pluginTestMutex.Unlock()

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test Plugin",
		"subcommand": "GetTsigKey",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(appLog)
	testRegistrar, err := NewPluginRegistrar(ctx, config)
	assert.Nil(t, err)

	// Test plugin will take are and place in Result
	testRegistrar.pluginArgs.PluginArg = map[string]interface{}{"FuncOutput": &dns.TSIGKey{Name: "tsig", Algorithm: "abcd", Secret: "boo"}}
	testSetup(testRegistrar)
	appLog.Debugf("Invoking %s library GetTsigKey", testRegistrar.pluginConfig.PluginName)
	testRegistrar.GetTsigKey(ctx, "test1.com")
	assert.Nil(t, testRegistrar.pluginResult.PluginError)
	key, _ := testRegistrar.pluginResult.PluginResult.(*dns.TSIGKey)
	assert.Equal(t, key.Name, "tsig")

}

func TestRegistrarGetTsigKeyFail(t *testing.T) {

	pluginTestMutex.Lock()
	defer pluginTestMutex.Unlock()

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test Plugin",
		"subcommand": "GetTsigKeyFail",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(appLog)
	testRegistrar, err := NewPluginRegistrar(ctx, config)
	assert.Nil(t, err)

	// Test plugin will take are and place in Result
	testRegistrar.pluginArgs.PluginArg = map[string]interface{}{"FuncErrors": "GetTsigKeyFail"}
	testSetup(testRegistrar)
	appLog.Debugf("Invoking %s library GetTsigKey", testRegistrar.pluginConfig.PluginName)
	testRegistrar.GetTsigKey(ctx, "test1.com")
	assert.NotNil(t, testRegistrar.pluginResult.PluginError)
	assert.Contains(t, testRegistrar.pluginResult.PluginError.Error(), "Fail")

}

func TestRegistrarGetServeAlgorithm(t *testing.T) {

	pluginTestMutex.Lock()
	defer pluginTestMutex.Unlock()

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test Plugin",
		"subcommand": "GetServeAlgorithm",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(appLog)
	testRegistrar, err := NewPluginRegistrar(ctx, config)
	assert.Nil(t, err)

	// Test plugin will take are and place in Result
	testRegistrar.pluginArgs.PluginArg = map[string]interface{}{"FuncOutput": "ServeAlgorithm"}
	testSetup(testRegistrar)
	appLog.Debugf("Invoking %s library GetServeAlgorithm", testRegistrar.pluginConfig.PluginName)
	testRegistrar.GetServeAlgorithm(ctx, "test1.com")
	assert.Nil(t, testRegistrar.pluginResult.PluginError)
	algo, _ := testRegistrar.pluginResult.PluginResult.(string)
	assert.Equal(t, algo, "ServeAlgorithm")
}

func TestRegistrarGetServeAlgorithmFail(t *testing.T) {

	pluginTestMutex.Lock()
	defer pluginTestMutex.Unlock()

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test Plugin",
		"subcommand": "GetServeAlgorithmFail",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(appLog)
	testRegistrar, err := NewPluginRegistrar(ctx, config)
	assert.Nil(t, err)

	// Test plugin will take are and place in Result
	testRegistrar.pluginArgs.PluginArg = map[string]interface{}{"FuncErrors": "GetServeAlgorithmFail"}
	testSetup(testRegistrar)
	appLog.Debugf("Invoking %s library GetServeAlgorithm", testRegistrar.pluginConfig.PluginName)
	testRegistrar.GetServeAlgorithm(ctx, "test1.com")
	assert.NotNil(t, testRegistrar.pluginResult.PluginError)
	assert.Contains(t, testRegistrar.pluginResult.PluginError.Error(), "Fail")

}

func TestRegistrarGetMasterIPs(t *testing.T) {

	pluginTestMutex.Lock()
	defer pluginTestMutex.Unlock()

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test Plugin",
		"subcommand": "GetMasterIPs",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(appLog)
	testRegistrar, err := NewPluginRegistrar(ctx, config)
	assert.Nil(t, err)

	// Test plugin will take are and place in Result
	testRegistrar.pluginArgs.PluginArg = map[string]interface{}{"FuncOutput": []string{"1.2.3.4", "4.5.6.7.8"}}
	testSetup(testRegistrar)
	appLog.Debugf("Invoking %s library GetMasterIPs", testRegistrar.pluginConfig.PluginName)
	testRegistrar.GetMasterIPs(ctx)
	assert.Nil(t, testRegistrar.pluginResult.PluginError)
	ml, _ := testRegistrar.pluginResult.PluginResult.([]string)
	assert.Equal(t, ml, []string{"1.2.3.4", "4.5.6.7.8"})

}

func TestRegistrarGetMasterIPsFail(t *testing.T) {

	pluginTestMutex.Lock()
	defer pluginTestMutex.Unlock()

	ctx := context.TODO()
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)

	appLog := log.WithFields(log.Fields{
		"registrar":  "Test Plugin",
		"subcommand": "GetMasterIPsFail",
	})
	ctx = context.WithValue(ctx, "appLog", appLog)

	config := initRegistrarStub(appLog)
	testRegistrar, err := NewPluginRegistrar(ctx, config)
	assert.Nil(t, err)

	// Test plugin will take are and place in Result
	testRegistrar.pluginArgs.PluginArg = map[string]interface{}{"FuncErrors": "GetMasterIPsFail"}
	testSetup(testRegistrar)
	appLog.Debugf("Invoking %s library GetMasterIPs", testRegistrar.pluginConfig.PluginName)
	testRegistrar.GetMasterIPs(ctx)
	assert.NotNil(t, testRegistrar.pluginResult.PluginError)
	assert.Contains(t, testRegistrar.pluginResult.PluginError.Error(), "Fail")
}
