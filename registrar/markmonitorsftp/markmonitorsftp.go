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
	dns "github.com/akamai/AkamaiOPEN-edgegrid-golang/configdns-v2"
	"github.com/akamai/edgedns-registrar-coordinator/registrar"
	log "github.com/apex/log"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"bufio"
	"context"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	//dfault cert algo
	DefaultCertAlgorithm = ""
	// default signature
	DefaultSignature = ""
	// default port
	DefaultHostPort = 22
	DefaultFileTTL  = time.Second * 600
)

var ()

// edgeDNSClient is a proxy interface of the MarkMonitor edgegrid configdns-v2 package that can be stubbed for testing.
type MarkMonitorDNSService interface {
	GetDomains(ctx context.Context) ([]string, error)
	GetDomain(ctx context.Context, domain string) (*registrar.Domain, error)
	GetTsigKey(ctx context.Context, domain string) (*dns.TSIGKey, error)
	GetServeAlgorithm(ctx context.Context, domain string) (string, error)
	GetMasterIPs(ctx context.Context) ([]string, error)
}

type SFTPDNSService interface {
	EstablishSFTPSession(log *log.Entry, markmonitorConfig *MarkMonitorSFTPConfig) error
	ReadRemoteDomainFile(log *log.Entry, markmonitorConfig *MarkMonitorSFTPConfig) (*[]string, error)
	ParseDomainFile(log *log.Entry, domFile *os.File) (*[]string, error)
}

type SFTPDNSConfig struct {
	sftpClient        *sftp.Client
	sshClient         *ssh.Client
	domainMasterList  *[]string
	lastDomFileUpdate time.Time
	domFileTTL        time.Duration
}

// MarkMonitorSFTPRegistrar implements the DNS registrar for Mark Monitor SFTP.
type MarkMonitorSFTPRegistrar struct {
	registrar.BaseRegistrarProvider
	markmonitorConfig *MarkMonitorSFTPConfig
	closeSFTPSession  func(interface{})
	// Defines client. Allows for mocking.
	sftpService SFTPDNSService
}

type MarkMonitorSFTPConfig struct {
	MarkMonitorSFTPConfigPath       string
	MarkMonitorSshUser              string   `yaml:"markmonitor_ssh_user"`
	MarkMonitorSshPassword          string   `yaml:"markmonitor_ssh_password"`
	MarkMonitorSshHost              string   `yaml:"markmonitor_ssh_host"`
	MarkMonitorSshPort              int      `yaml:"markmonitor_ssh_port"`
	MarkMonitorSslCertAlgorithm     string   `yaml:"markmonitor_ssl_cert_algorithm"`
	MarkMonitorSslSignature         string   `yaml:"markmonitor_ssl_signature"`
	MarkMonitorSftpPktSize          int      `yaml:"markmonitor_sftp_pkt_size"`
	MarkMonitorMasterIPs            []string `yaml:"markmonitor_master_ips"`
	MarkMonitorDomainConfigFilePath string   `yaml:"markmonitor_registrar_domain_filepath"`
	MarkMonitorTempDomainFileFolder string   `yaml:"markmonitor_temp_file_folder"`
	MarkMonitorDomFileTTL           string   `yaml:"markmonitor_domain_file_ttle"` // in seconds
}

// Create and return ssl Connection
func initSSHClient(log *log.Entry, markmonitorConfig *MarkMonitorSFTPConfig) (*ssh.Client, error) {

	// use known_hosts in the users home directory
	hostKeyCallback, err := knownhosts.New(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		log.Errorf("could not create hostkeycallback function: %s", err.Error())
		return nil, err
	}
	config := &ssh.ClientConfig{
		User: markmonitorConfig.MarkMonitorSshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(markmonitorConfig.MarkMonitorSshPassword),
		},
		HostKeyCallback: hostKeyCallback,
	}
	if markmonitorConfig.MarkMonitorSslCertAlgorithm != "" {
		config.HostKeyAlgorithms = append(config.HostKeyAlgorithms, markmonitorConfig.MarkMonitorSslCertAlgorithm)
	}

	sshAddr := markmonitorConfig.MarkMonitorSshHost + ":" + strconv.Itoa(markmonitorConfig.MarkMonitorSshPort)
	// connect
	client, err := ssh.Dial("tcp", sshAddr, config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Create and return sftp client
func initSFTPClient(log *log.Entry, markmonitorConfig *MarkMonitorSFTPConfig, sshClient *ssh.Client) (*sftp.Client, error) {

	var err error
	var sftpClient *sftp.Client
	// create new SFTP client
	if markmonitorConfig.MarkMonitorSftpPktSize != 0 {
		sftpClient, err = sftp.NewClient(sshClient, sftp.MaxPacket(markmonitorConfig.MarkMonitorSftpPktSize))
	} else {
		sftpClient, err = sftp.NewClient(sshClient)
	}
	if err != nil {
		return nil, err
	}

	return sftpClient, nil

}

func closeSFTPSession(config interface{}) {

	sftpDNSConfig := config.(*SFTPDNSConfig)
	// close active connectios
	if sftpDNSConfig.sftpClient != nil {
		sftpDNSConfig.sftpClient.Close()
	}
	if sftpDNSConfig.sshClient != nil {
		sftpDNSConfig.sshClient.Close()
	}
	// clear
	sftpDNSConfig.sftpClient = nil
	sftpDNSConfig.sshClient = nil
}

// NewMarkMonitorProvider initializes a new MarkMonitor DNS based Provider.
func NewMarkMonitorSFTPRegistrar(ctx context.Context, mmConfig MarkMonitorSFTPConfig, sftpService SFTPDNSService) (*MarkMonitorSFTPRegistrar, error) {

	var err error

	log := ctx.Value("appLog").(*log.Entry)
	markmonitorConfig := &mmConfig
	// if mock, skip
	if sftpService == nil {
		// Get file config and parse
		if mmConfig.MarkMonitorSFTPConfigPath == "" {
			return nil, fmt.Errorf("MarkMonitor Registrar requires a configuration file")
		}
		markmonitorConfig, err = loadConfig(log, mmConfig.MarkMonitorSFTPConfigPath)
		if err != nil {
			return nil, fmt.Errorf("MarkMonitor Registrar. Invalid configuration file")
		}
	}
	// Set up ssl and sftp clients/session
	if markmonitorConfig.MarkMonitorSshHost == "" || markmonitorConfig.MarkMonitorSshUser == "" || markmonitorConfig.MarkMonitorSshPassword == "" {
		return nil, fmt.Errorf("MarkMonitor Registrar. Invalid configuration file. One or more required credentials missing.")
	}
	if len(markmonitorConfig.MarkMonitorMasterIPs) < 1 {
		return nil, fmt.Errorf("MarkMonitor Registrar. One or more Master IPs required.")
	}
	if markmonitorConfig.MarkMonitorDomainConfigFilePath == "" {
		return nil, fmt.Errorf("MarkMonitor Registrar. Invalid configuration file. Remote domain file path missing.")
	}
	if markmonitorConfig.MarkMonitorSslCertAlgorithm == "" {
		log.Infof("MarkMonitor using default SSL Certificate Algorithm: %s", DefaultCertAlgorithm)
		markmonitorConfig.MarkMonitorSslCertAlgorithm = DefaultCertAlgorithm
	}
	if markmonitorConfig.MarkMonitorSslSignature == "" {
		log.Infof("MarkMonitor using default SSL Signature: %s", DefaultSignature)
		markmonitorConfig.MarkMonitorSslSignature = DefaultSignature
	}
	if markmonitorConfig.MarkMonitorSshPort == 0 {
		log.Infof("MarkMonitor using default port: %v", DefaultHostPort)
		markmonitorConfig.MarkMonitorSshPort = DefaultHostPort
	}

	provider := &MarkMonitorSFTPRegistrar{
		markmonitorConfig: markmonitorConfig,
		sftpService:       &SFTPDNSConfig{},
		closeSFTPSession:  closeSFTPSession,
	}
	if sftpService != nil {
		log.Debugf("Using STUB")
		provider.sftpService = sftpService
	} else {
		err := provider.sftpService.EstablishSFTPSession(log, markmonitorConfig)
		defer closeSFTPSession(provider.sftpService)
		if err != nil {
			log.Errorf("MarkMonitor Registrar. Failed to initialize SFTP Client. %s", err.Error())
			return nil, fmt.Errorf("MarkMonitor Registrar. Failed to initialize SFTP Client.")
		}
		dur, err := time.ParseDuration(markmonitorConfig.MarkMonitorDomFileTTL)
		if provider.markmonitorConfig.MarkMonitorDomFileTTL != "" && err == nil {
			provider.sftpService.(*SFTPDNSConfig).domFileTTL = dur
		} else {
			provider.sftpService.(*SFTPDNSConfig).domFileTTL = DefaultFileTTL
		}
	}

	return provider, nil
}

//

func closeAndRemoveDomainsFile(log *log.Entry, localDomsFile *os.File) {

	// if can't defer in defer, will need to place inline
	defer os.Remove(localDomsFile.Name())
	if err := localDomsFile.Close(); err != nil {
		log.Warnf("Failed to close temp file. %s", err.Error())
	}

	return
}

// parseDomainsConfigFile parses MM domains file into a list.
func parseDomainsConfigFile(localDomsFile *os.File) ([]string, error) {

	// STUB
	return []string{"zone-1.com", "zone-2.com"}, nil

}

/*
Service Entry Points
*/

func (mm *MarkMonitorSFTPRegistrar) GetDomains(ctx context.Context) ([]string, error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering MarkMonitor registrar GetDomains")

	defer mm.closeSFTPSession(mm.sftpService)
	err := mm.sftpService.EstablishSFTPSession(log, mm.markmonitorConfig)
	if err != nil {
		log.Errorf(" MarkMonitor GetDomains: Failed to initialize SFTP Client. %s", err.Error())
		return []string{}, fmt.Errorf("MarkMonitor GetDomains: Failed to initialize SFTP Client.")
	}

	domains, err := mm.sftpService.ReadRemoteDomainFile(log, mm.markmonitorConfig)
	if err != nil {
		log.Errorf(" MarkMonitor GetDomains: Failed. %s", err.Error())
		return []string{}, fmt.Errorf("MarkMonitor GetDomains: Failed to parse domains file.")
	}

	log.Debugf("Registrar GetDomains result: %v", domains)

	return *domains, nil
}

func (mm *MarkMonitorSFTPRegistrar) GetDomain(ctx context.Context, domain string) (*registrar.Domain, error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering MarkMonitor registrar GetDomain")

	defer mm.closeSFTPSession(mm.sftpService)
	err := mm.sftpService.EstablishSFTPSession(log, mm.markmonitorConfig)
	if err != nil {
		log.Errorf(" MarkMonitor GetDomains: Failed to initialize SFTP Client. %s", err.Error())
		return nil, fmt.Errorf("MarkMonitor GetDomains: Failed to initialize SFTP Client.")
	}

	zone := "not implemented"
	log.Debugf("Registrar GetDomain result: %v", zone)

	/*
		return &registrar.Domain{
			Name:                  zone.Zone,
			Type:                  zone.Type,
			SignAndServe:          zone.SignAndServe,
			SignAndServeAlgorithm: zone.SignAndServeAlgorithm,
			Masters:               zone.Masters,
			TsigKey:               zone.TsigKey,
		}, nil
	*/

	return &registrar.Domain{}, nil
}

func (mm *MarkMonitorSFTPRegistrar) GetTsigKey(ctx context.Context, domain string) (tsigKey *dns.TSIGKey, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering MarkMonitor registrar GetTsigKey")

	defer mm.closeSFTPSession(mm.sftpService)
	err = mm.sftpService.EstablishSFTPSession(log, mm.markmonitorConfig)
	if err != nil {
		log.Errorf(" MarkMonitor GetDomains: Failed to initialize SFTP Client. %s", err.Error())
		return nil, fmt.Errorf("MarkMonitor GetDomains: Failed to initialize SFTP Client.")
	}

	log.Info("MarkMonitorSFTPRegistrar does not support Tsig")

	return
}

func (mm *MarkMonitorSFTPRegistrar) GetServeAlgorithm(ctx context.Context, domain string) (algo string, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering MarkMonitor registrar GetServeAlgorithm")

	defer mm.closeSFTPSession(mm.sftpService)
	err = mm.sftpService.EstablishSFTPSession(log, mm.markmonitorConfig)
	if err != nil {
		log.Errorf(" MarkMonitor GetDomains: Failed to initialize SFTP Client. %s", err.Error())
		return "", fmt.Errorf("MarkMonitor GetDomains: Failed to initialize SFTP Client.")
	}

	log.Info("MarkMonitorSFTPRegistrar does not support DNSSEC")

	return
}

func (mm *MarkMonitorSFTPRegistrar) GetMasterIPs(ctx context.Context) (masters []string, err error) {

	log := ctx.Value("appLog").(*log.Entry)
	log.Debug("Entering MarkMonitor registrar GetMasterIPs")

	log.Debugf("Registrar GetMasterIPs result: %v", mm.markmonitorConfig.MarkMonitorMasterIPs)
	return mm.markmonitorConfig.MarkMonitorMasterIPs, nil
}

//
// Config file processing
//
func loadConfig(log *log.Entry, configFile string) (*MarkMonitorSFTPConfig, error) {

	log.Debug("Entering MarkMonitor registrar loadConfig")
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

func loadConfigContent(log *log.Entry, configData []byte) (*MarkMonitorSFTPConfig, error) {
	config := MarkMonitorSFTPConfig{}
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

//
//  Stubbable functions
//

// establish SFTPSession if not already
func (s *SFTPDNSConfig) EstablishSFTPSession(log *log.Entry, markmonitorConfig *MarkMonitorSFTPConfig) error {

	if s.sshClient == nil {
		sshClient, err := initSSHClient(log, markmonitorConfig)
		if err != nil {
			log.Errorf("MarkMonitor Registrar. Failed to initialize SSH Client. %s", err.Error())
			return fmt.Errorf("MarkMonitor Registrar. Failed to initialize SSH Client.")
		}
		s.sshClient = sshClient
	}
	if s.sftpClient == nil {
		sftpClient, err := initSFTPClient(log, markmonitorConfig, s.sshClient)
		if err != nil {
			log.Errorf("MarkMonitor Registrar. Failed to initialize SFTP Client. %s", err.Error())
			return fmt.Errorf("MarkMonitor Registrar. Failed to initialize SFTP Client.")
		}
		s.sftpClient = sftpClient
	}

	return nil
}

func ParseZoneData(log *log.Entry, zoneLine string) string {

	log.Debugf("Domains line: [%s]", zoneLine)
	if !strings.HasPrefix(zoneLine, "zone") {
		return ""
	}
	if !strings.Contains(zoneLine, "slave") {
		log.Debugf("Skipping zone line [%s]", zoneLine)
		return ""
	}
	dline := strings.SplitN(zoneLine, " ", 4)
	if len(dline) < 4 {
		log.Warnf("Incomplete zone line: %s", zoneLine)
		return ""
	}
	zoneText := strings.Split(dline[1], "\"")

	return zoneText[1]

}

// ParseDomainFile parses reteieved domains file. Returns map of domains indexed by masterp ip and error
func (s *SFTPDNSConfig) ParseDomainFile(log *log.Entry, domFile *os.File) (*[]string, error) {

	log.Debugf("Entering ParseDomainFile")
	// File line example (excluding preamble and postables):
	// zone "genevarx.com" in { type slave; file "/var/dns-config/dbs/zone.genevarx.com.bak"; masters { 64.124.14.39; }; allow-transfer {def_xfer; }; };

	defer closeAndRemoveDomainsFile(log, domFile)

	domNames := []string{}
	// start from beginning of the file
	if _, err := domFile.Seek(0, 0); err != nil {
		return &domNames, err
	}
	scanner := bufio.NewScanner(domFile)
	for scanner.Scan() {
		zoneLine := strings.TrimSpace(scanner.Text())
		zone := ParseZoneData(log, zoneLine)
		if zone != "" {
			log.Debugf("Adding %s to domain list", zone)
			domNames = append(domNames, zone)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Errorf(err.Error())
		return nil, err
	}

	return &domNames, nil

}

// ReadRemoteDomainFile reads remote dmains file, saves to remp location. returns handle of temp file and error.
func (s *SFTPDNSConfig) ReadRemoteDomainFile(log *log.Entry, markmonitorConfig *MarkMonitorSFTPConfig) (*[]string, error) {

	log.Debugf("Entering ReadRemoteDomainFile")

	fstat, err := s.sftpClient.Stat(markmonitorConfig.MarkMonitorDomainConfigFilePath)
	if err != nil {
		log.Errorf("ReadRemoteDomainFile: Failed to stat remote domains file. %s", err.Error())
		return nil, err
	}

	modTime := fstat.ModTime()
	if modTime.After(s.lastDomFileUpdate.Add(s.domFileTTL)) && s.domainMasterList != nil {
		return s.domainMasterList, nil
	}

	// open remote domains file
	domsFile, err := s.sftpClient.Open(markmonitorConfig.MarkMonitorDomainConfigFilePath)
	if err != nil {
		log.Errorf("ReadRemoteDomainFile: Failed to open remote domains file. %s", err.Error())
		return nil, err
	}
	defer domsFile.Close()

	// create temporary file
	tempFile, err := ioutil.TempFile(markmonitorConfig.MarkMonitorTempDomainFileFolder, "MMDomainConfig-*")
	if err != nil {
		log.Errorf("ReadRemoteDomainFile: Failed to create temp domains file. %s", err.Error())
		return nil, err
	}
	// copy domains file to temp file
	_, err = io.Copy(tempFile, domsFile)
	if err != nil {
		closeAndRemoveDomainsFile(log, tempFile)
		log.Errorf("ReadRemoteDomainFile: Failed to copy file from remote. %s", err.Error())
		return nil, err
	}

	// flush in-memory copy
	err = tempFile.Sync()
	if err != nil {
		closeAndRemoveDomainsFile(log, tempFile)
		log.Errorf("ReadRemoteDomainFile: Failed to persist copied file. %s", err.Error())
		return nil, err
	}

	domsList, err := s.ParseDomainFile(log, tempFile)
	if err != nil {
		return nil, err
	}

	s.domainMasterList = domsList
	s.lastDomFileUpdate = modTime

	return domsList, nil
}
