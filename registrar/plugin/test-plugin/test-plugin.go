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
	"fmt"
	"github.com/akamai/edgedns-registrar-coordinator/registrar"
	log "github.com/apex/log"
)

const ()

var (
	LibPluginArgs   registrar.PluginFuncArgs
	LibPluginResult registrar.PluginFuncResult
	libLog          *log.Entry
)

// NewPluginProvider initializes a new test registrar plugin lib.
func NewPluginLibRegistrar() {

	pluginConfig := LibPluginArgs.PluginArg.(registrar.PluginConfig)
	libLog = pluginConfig.LogEntry

	return
}

/*
	Test harness passes in map[string]interface{} with FuncOutput and FuncErrors entries

*/

func GetDomains() {

	libLog.Debug("Entering Test Plugin Lib registrar GetDomains")
	pluginObj := LibPluginArgs.PluginArg.(map[string]interface{})

	if output, ok := pluginObj["FuncOutput"]; ok {
		LibPluginResult.PluginResult = output
	}
	if errmsg, ok := pluginObj["FuncErrors"]; ok {
		err := fmt.Errorf("GetDomains Failed. %s", errmsg.(string))
		LibPluginResult.PluginError = err
	}

	return
}

func GetDomain() {

	libLog.Debug("Entering Test Plugin Lib registrar GetDomain")

	pluginObj := LibPluginArgs.PluginArg.(map[string]interface{})
	if output, ok := pluginObj["FuncOutput"]; ok {
		LibPluginResult.PluginResult = output
	}
	if errmsg, ok := pluginObj["FuncErrors"]; ok {
		err := fmt.Errorf("GetDomain Failed. %s", errmsg.(string))
		LibPluginResult.PluginError = err
	}

	return
}

func GetTsigKey() {

	libLog.Debug("Entering Test Plugin Lib registrar GetTsigKey")

	pluginObj := LibPluginArgs.PluginArg.(map[string]interface{})
	if output, ok := pluginObj["FuncOutput"]; ok {
		LibPluginResult.PluginResult = output
	}
	if errmsg, ok := pluginObj["FuncErrors"]; ok {
		err := fmt.Errorf("GetTsigKey Failed. %s", errmsg.(string))
		LibPluginResult.PluginError = err
	}

	return
}

func GetServeAlgorithm() {

	libLog.Debug("Entering Test Plugin Lib registrar GetServeAlgorithm")

	pluginObj := LibPluginArgs.PluginArg.(map[string]interface{})
	if output, ok := pluginObj["FuncOutput"]; ok {
		LibPluginResult.PluginResult = output
	}
	if errmsg, ok := pluginObj["FuncErrors"]; ok {
		err := fmt.Errorf("GetServeAlgorithm Failed. %s", errmsg.(string))
		LibPluginResult.PluginError = err
	}

	return
}

func GetMasterIPs() {

	libLog.Debug("Entering Test Plugin Lib registrar GetMasterIPs")
	pluginObj := LibPluginArgs.PluginArg.(map[string]interface{})
	if output, ok := pluginObj["FuncOutput"]; ok {
		LibPluginResult.PluginResult = output
	}
	if errmsg, ok := pluginObj["FuncErrors"]; ok {
		err := fmt.Errorf("GetMasterIPs Failed. %s", errmsg.(string))
		LibPluginResult.PluginError = err
	}

	return
}

func main() {

	fmt.Println("Test Plugin Library Registrar")
}
