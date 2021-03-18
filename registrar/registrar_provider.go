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

package registrar

import (
	"context"
	dns "github.com/akamai/AkamaiOPEN-edgegrid-golang/configdns-v2"
	log "github.com/apex/log"
	"plugin"
)

const ()

var ()

type Domain struct {
	Name                  string
	Type                  string
	SignAndServe          bool
	SignAndServeAlgorithm string
	Masters               []string
	TsigKey               *dns.TSIGKey
}

// Integrated Registrar

type RegistrarProvider interface {
	GetDomains(ctx context.Context) ([]string, error)
	GetDomain(ctx context.Context, domain string) (*Domain, error)
	GetTsigKey(ctx context.Context, domain string) (*dns.TSIGKey, error)
	GetServeAlgorithm(ctx context.Context, domain string) (string, error)
	GetMasterIPs(ctx context.Context) ([]string, error)
	//GetTsigKeys() []dnsTSIGKey
	//GetDnsSecStatus
	//GetZoneTransferStatus
}

type BaseRegistrarProvider struct {
}

func (b BaseRegistrarProvider) GetDomains(ctx context.Context) (domains []string, err error) {

	return
}

func (b BaseRegistrarProvider) GetDomain(ctx context.Context, domain string) (*Domain, error) {

	return nil, nil
}

func (b BaseRegistrarProvider) GetTsigKey(ctx context.Context, domain string) (*dns.TSIGKey, error) {

	return nil, nil
}

func (b BaseRegistrarProvider) GetServeAlgorithm(ctx context.Context, domain string) (string, error) {

	return "", nil
}

func (b BaseRegistrarProvider) GetMasterIPs(ctx context.Context) (masterIps []string, err error) {

	return
}

// Plugins

type PluginRegistrarProvider interface {
	NewPluginLibRegistrar()
	GetDomains()
	GetDomain()
	GetTsigKey()
	GetServeAlgorithm()
	GetMasterIPs()
	//GetTsigKeys() []dnsTSIGKey
	//GetDnsSecStatus
	//GetZoneTransferStatus
}

type PluginConfig struct {
	PluginLibPath    string
	PluginName       string
	PluginConfigPath string
	LogEntry         *log.Entry
	Registrar        *plugin.Plugin
}

type PluginFuncArgs struct {
	PluginArg interface{}
}

type PluginFuncResult struct {
	PluginResult interface{}
	PluginError  error
}
