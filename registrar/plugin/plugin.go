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
	"context"
	"fmt"
	dns "github.com/akamai/AkamaiOPEN-edgegrid-golang/configdns-v2"
	"github.com/akamai/edgedns-registrar-coordinator/registrar"
	log "github.com/apex/log"
	"path/filepath"
	"plugin"
	"sync"
)

const ()

var (
	pluginMutex = &sync.Mutex{}
	// TODO Create mutex map with plugin name as index?
)

// Plugin Registrar
type PluginRegistrar struct {
	registrar.BaseRegistrarProvider
	pluginConfig            *registrar.PluginConfig
	pluginArgs              *registrar.PluginFuncArgs
	pluginResult            *registrar.PluginFuncResult
	pluginGetDomains        func()
	pluginGetDomain         func()
	pluginGetMasterIPs      func()
	pluginGetTsigKey        func()
	pluginGetServeAlgorithm func()
}

func lookupSymbols(plug *plugin.Plugin, reg *PluginRegistrar) error {

	sym, err := plug.Lookup("LibPluginArgs")
	if err != nil {
		log.Errorf("Plugin library does not support RegistrarProvider interface. Error: %s", err.Error())
		return err
	}
	reg.pluginArgs = sym.(*registrar.PluginFuncArgs)
	sym, err = plug.Lookup("LibPluginResult")
	if err != nil {
		log.Errorf("Plugin library does not support RegistrarProvider interface. Error: %s", err.Error())
		return err
	}
	reg.pluginResult = sym.(*registrar.PluginFuncResult)
	sym, err = plug.Lookup("GetDomains")
	if err != nil {
		log.Errorf("Plugin library does not support RegistrarProvider interface. Error: %s", err.Error())
		return err
	}
	/*
	   if _, ok := sym.(registrar.PluginRegistrarProvider); !ok {
	           log.Errorf("Plugin library failed validation. Error: %s", err.Error())
	           return err
	   }
	*/
	reg.pluginGetDomains = sym.(func())
	sym, err = plug.Lookup("GetDomain")
	if err != nil {
		log.Errorf("Plugin library does not support RegistrarProvider interface. Error: %s", err.Error())
		return err
	}
	reg.pluginGetDomain = sym.(func())
	sym, err = plug.Lookup("GetTsigKey")
	if err != nil {
		log.Errorf("Plugin library does not support RegistrarProvider interface. Error: %s", err.Error())
		return err
	}
	reg.pluginGetTsigKey = sym.(func())
	sym, err = plug.Lookup("GetServeAlgorithm")
	if err != nil {
		log.Errorf("Plugin library does not support RegistrarProvider interface. Error: %s", err.Error())
		return err
	}
	reg.pluginGetServeAlgorithm = sym.(func())
	sym, err = plug.Lookup("GetMasterIPs")
	if err != nil {
		log.Errorf("Plugin library does not support RegistrarProvider interface. Error: %s", err.Error())
		return err
	}
	reg.pluginGetMasterIPs = sym.(func())

	return nil

}

// NewPluginRegistrar initializes a new plugin registrar
func NewPluginRegistrar(ctx context.Context, pluginConfig registrar.PluginConfig) (*PluginRegistrar, error) {

	var err error

	pluginMutex.Lock()
	defer pluginMutex.Unlock()

	log := ctx.Value("appLog").(*log.Entry)
	log.Debugf("Entering NewPluginRegistrar")
	pluginRegistrar := PluginRegistrar{pluginConfig: &pluginConfig}
	// Parse validation should ensure path is not empty
	regPlugin, err := plugin.Open(pluginConfig.PluginLibPath)
	if err != nil {
		log.Errorf("Failed to open provided plugin library. Error: %s", err.Error())
		return nil, err
	}
	// Get plugin in name
	pluginConfig.PluginName = filepath.Base(pluginConfig.PluginLibPath)

	if err = lookupSymbols(regPlugin, &pluginRegistrar); err != nil {
		log.Errorf("Plugin library failed validation. Error: %s", err.Error())
		return nil, err
	}

	// initialize the plugin
	newPluginLibRegistrar, err := regPlugin.Lookup("NewPluginLibRegistrar")
	if err != nil {
		log.Errorf("Plugin library does not support RegistrarProvider interface. Error: %s", err.Error())
		return nil, err
	}
	pluginRegistrar.pluginArgs.PluginArg = pluginConfig
	newPluginLibRegistrar.(func())()
	if pluginRegistrar.pluginResult.PluginError != nil {
		log.Errorf("Plugin library failed to initialize. Error: %s", pluginRegistrar.pluginResult.PluginError.Error())
		return nil, pluginRegistrar.pluginResult.PluginError
	}
	pluginConfig.Registrar = regPlugin

	return &pluginRegistrar, nil
}

func (r *PluginRegistrar) GetDomains(ctx context.Context) ([]string, error) {

	var domainsList = []string{}

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering Plugin registrar GetDomains")
	// Synchronize library calls
	pluginMutex.Lock()
	defer pluginMutex.Unlock()

	log.Debugf("Invoking %s library GetDomains", r.pluginConfig.PluginName)
	r.pluginGetDomains()
	if r.pluginResult.PluginError != nil {
		log.Errorf("Plugin library GetDomains failed. %s", r.pluginResult.PluginError.Error())
		return domainsList, fmt.Errorf("Plugin library GetDomains failed.")
	}
	dl, ok := r.pluginResult.PluginResult.([]string)
	if !ok {
		log.Debugf("Unexpected Plugin library GetDomains return value: %v", r.pluginResult.PluginResult)
		return domainsList, fmt.Errorf("Unexpected Plugin library GetDomains return value type")
	}
	for _, d := range dl {
		domainsList = append(domainsList, d)
	}

	log.Debugf("Plugin GetDomains result: %v", domainsList)
	return domainsList, nil

}

func (r *PluginRegistrar) GetDomain(ctx context.Context, domain string) (*registrar.Domain, error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering Plugin registrar GetDomain")
	// Synchronize library calls
	pluginMutex.Lock()
	defer pluginMutex.Unlock()

	log.Debugf("Invoking %s library GetDomain", r.pluginConfig.PluginName)
	r.pluginArgs.PluginArg = domain
	r.pluginGetDomain()
	if r.pluginResult.PluginError != nil {
		return nil, r.pluginResult.PluginError
	}
	log.Debugf("Plugin GetDomain result: %v", r.pluginResult.PluginResult)
	libDom, ok := r.pluginResult.PluginResult.(registrar.Domain)
	if !ok {
		return nil, fmt.Errorf("Unexpected Plugin library GetDomain return value type")
	}
	return &registrar.Domain{
		Name:                  libDom.Name,
		Type:                  libDom.Type,
		SignAndServe:          libDom.SignAndServe,
		SignAndServeAlgorithm: libDom.SignAndServeAlgorithm,
		Masters:               libDom.Masters,
		TsigKey:               libDom.TsigKey,
	}, nil
}

func (r *PluginRegistrar) GetTsigKey(ctx context.Context, domain string) (tsigKey *dns.TSIGKey, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering Plugin registrar GetTsigKey")
	// Synchronize library calls
	pluginMutex.Lock()
	defer pluginMutex.Unlock()

	log.Debugf("Invoking %s library GetTsigKey", r.pluginConfig.PluginName)
	r.pluginArgs.PluginArg = domain
	r.pluginGetTsigKey()
	if r.pluginResult.PluginError != nil {
		return nil, r.pluginResult.PluginError
	}
	libTsig, ok := r.pluginResult.PluginResult.(dns.TSIGKey)
	if !ok {
		log.Debugf("Unexpected Plugin library GetTsigKey return value: %v", r.pluginResult.PluginResult)
		return nil, fmt.Errorf("Unexpected Plugin library GetTsigKey return value type")
	}
	tsigKey.Name = libTsig.Name
	tsigKey.Algorithm = libTsig.Algorithm
	tsigKey.Secret = libTsig.Secret

	log.Debugf("Returning Registrar GetTsigKey result")
	return
}

func (r *PluginRegistrar) GetServeAlgorithm(ctx context.Context, domain string) (algo string, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering Plugin registrar GetServeAlgorithm")
	// Synchronize library calls
	pluginMutex.Lock()
	defer pluginMutex.Unlock()

	log.Debugf("Invoking %s library GetServeAlgorithm", r.pluginConfig.PluginName)
	r.pluginArgs.PluginArg = domain
	r.pluginGetServeAlgorithm()
	if r.pluginResult.PluginError != nil {
		return "", r.pluginResult.PluginError
	}

	algo, ok := r.pluginResult.PluginResult.(string)
	if !ok {
		log.Debugf("Unexpected Plugin library GetServeAlgorithm return value: %v", r.pluginResult.PluginResult)
		return "", fmt.Errorf("Unexpected Plugin library GetServeAlgorithm return value type")
	}
	log.Debugf("Returning Registrar GetServeAlgorithm result")
	return
}

func (r *PluginRegistrar) GetMasterIPs(ctx context.Context) ([]string, error) {

	var masters = []string{}

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering Akamai registrar GetMasterIPs")
	// Synchronize library calls
	pluginMutex.Lock()
	defer pluginMutex.Unlock()

	log.Debugf("Invoking %s library GetMasterIPs", r.pluginConfig.PluginName)
	r.pluginGetMasterIPs()
	if r.pluginResult.PluginError != nil {
		return masters, r.pluginResult.PluginError
	}

	mlist, ok := r.pluginResult.PluginResult.([]string)
	if !ok {
		log.Debugf("Unexpected Plugin library GetMasterIPs return value: %v", r.pluginResult.PluginResult)
		return masters, fmt.Errorf("Unexpected Plugin library GetMasterIPs return value type")
	}
	log.Debugf("Plugin GetMasterIPs result: %v", mlist)
	masters = append(masters, mlist...)
	return masters, nil
}
