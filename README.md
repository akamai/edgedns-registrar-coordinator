# edgedns-registrar-coordinator

This library is a technical preview of the Akamai Edge DNS Registrar Coordinator. The current version of the Coordinator synchronizes Primary zones on a target registrar with Secondary zones in Akamai Edge DNS.

This release supports Akamai, Plugin and Mark Monitor SFTP registrars.

## Getting Started

The Coordinator can be installed by retrieving the appropriate binary from [github](https://github.com/akamai/edgedns-registrar-coordinator/releases) or by getting and building the binary locally. Binaries are available for linux386, linuxamd64, macamd64, windows386, and windowsamd64.

## Prerequisites

* Valid API client with authorization to use the Global Traffic Management Reporting API. [Akamai API Authentication](https://developer.akamai.com/getting-started/edgegrid) provides an overview and further information pertaining to the generation of authorization credentials for API based applications and tools.
* If building locally:
  * [Go environment](https://golang.org/doc/install)
  * [Git source management](https://git-scm.com/downloads)
  * [GNU Make](https://www.gnu.org/software/make/)

## Install Binary

1. [Download](https://github.com/akamai/edgedns-registrar-coordinator/releases) the appropriate binary for your platform.
2. Place the binary in the location you want to run from.
3. Launch the binary with the appropriate command line arguments.

## Build Locally

To build the Coordinator locally, follow these steps: 

```bash
$ git clone github.com/akamai/edgedns-registrar-coordinator
$ cd edgedns-registrar-coordinator
$ ./build.sh $(echo $(grep "VERSION" ./main.go | cut -d"=" -f2- | tr -d \''"\' 2> /dev/null)) 
```

## Coordinator Usage

The command line help itemizes valid command line arguments for the Coordinator.

```
$ build/edgedns-registrar-coordinator-0.1.0-linuxamd64 --help
ParseFlags Command Args:  [--help]
usage: edgedns-registrar-coordinator --registrar=REGISTRAR [<flags>] <command> [<args> ...]

A command-line application for coordination of registrar actions with Akamai Edge DNS.

Note that all flags may be replaced with env vars - `--registrar` -> `EDGEDNS-REGISTRAR-COORDINATOR-REGISTRAR` or `--registrar value` ->
`EDGEDNS-REGISTRAR-COORDINATOR-REGISTRAR=value`

Flags:
  --help                         Show context-sensitive help (also try --help-long and --help-man).
  --registrar=REGISTRAR          registrar
  --registrar-config-path=REGISTRAR-CONFIG-PATH
                                 registrar configuration filepath
  --interval=15m0s               registrar coordination interval in duration format (default: 15m)
  --dnssec                       Enables DNSSEC Serve (default: disabled)
  --tsig                         Enables TSIG Key processing (default: disabled)
  --once                         When enabled, exits the synchronization loop after the first iteration (default: disabled)
  --dry-run                      When enabled, prints DNS record changes rather than actually performing them (default: disabled)
  --log-file-path=""             The log file path. Default destination is stderr
  --log-handler=text             The handler used to log messages (default: text. options: text, json, cli, discard)
  --log-level=info               Set the level of logging (default: info, options: debug, info, warning, error, fatal)
  --edgedns-contract=""          Contract to use creating a domain
  --edgedns-group=EDGEDNS-GROUP  group id to use creating a domain
  --edgegrid-host=""             EdgeDNS API Server URL
  --edgegrid-client-token=""     EdgeDNS API Client Token
  --edgegrid-client-secret=""    EdgeDNS API Client Secret
  --edgegrid-access-token=""     EdgeDNS API Access Token
  --edgegrid-edgerc-path=""      optionally specify the .edgerc file path instead of individual Edgegrid keys
  --edgegrid-edgerc-section=""   specify the section when specifying an .edgerc file path
  --plugin-filepath=""           plugin provider library location path.

Commands:
  help [<command>...]
    Show help.

  monitor
    Monitor registrar for domain adds and deletes.
$
```

An example invocation utilizing the Akamai registrar would be:
```
$ ./edgedns-registrar-coordinator monitor --registrar akamai --interval 5m --edgegrid-edgerc-path /home/testuser/.edgerc --edgedns-contract 1-ABCDE9 --edgedns-group 12345 --registrar-config-path ./akamai-registrar-config.yaml --log-level debug --log-file-path ./test.log --once --dry-run 
```

where the configuration file contains:

```
akamai_contracts: 1-5C13O2

akamai_name_filter: edgedns

#akamai_edgerc_path: /home/test/.edgerc

#akamai_edgerc_section: registrar

akamai_host: akab-akamairegistrarh-ostexample123456.luna.akamaiapis.net
  
akamai_access_token: akab-akamaiaccesstoke-nexample12345678
  
akamai_client_token: akab-akamaiclienttoke-nexample12345678

akamai_client_secret: akamaiclientsecretexampleforregistrar123456=
```

An example invocation utilizing the Plugin registrar and Akamai plugin library would be:
```
./edgedns-registrar-coordinator monitor --registrar plugin --interval 5m --dry-run --edgegrid-edgerc-path /home/testuser/.edgerc --edgedns-contract 1-ABCDE1 --edgedns-group 98765 --plugin-filepath  /home/github.com/akamai/edgedns-registrar-coordinator/registrar/akamai-plugin-lib/akamai-plugin-lib --registrar-config-path ./akamai-plugin-lib-config.yaml --log-level debug
```

where the configuration file contains:

```
akamai_contracts: 1-ABC123

akamai_name_filter: edgedns

#akamai_edgerc_path: /home/test/.edgerc

#akamai_edgerc_section: registrar

akamai_host: akab-abcdefghijklmnopqrstuvwxyz1234567.luna.akamaiapis.net
  
akamai_access_token: akab-exampleaccesstok-en1234567890abcd
  
akamai_client_token: akab-exampleaccesstok-en12345678901234

akamai_client_secret: exampleakamaiclientsecret1234567890abcdefgh

#akamai_client_maxbody:
  
#akamai_client_account_key:
```

## Sub Commands

The current release of the Akamai Edge DNS Registrar Coordinator exposes a single sub command, `monitor`, that synchronizes the target registrar and Edge DNS. The monitor sub command requires edgegrid credentials, contract and group information. Registrars are initialized based on a provided config file as necessary for each registrar. Monitor retrieves the list of primary domains from the registrar and secondary domains from Edge DNS, ensuring that there is a pairing for each domain name in the registrar list. If not, the monitor process reconciles by removing secondary domains from Edge DNS that are no longer represented in the registrar, as well as creating secondary domains which are not present in Edge DNS.

## Registrars

The current release of the Akamai Edge DNS Registrar Coordinator supports three registrars, `akamai`, `plugin` and `markmonitorsftp`. 

### Akamai Registrar

The Akamai registrar serves as an example for the development of additional registrars as well as demonstrating the operation of the monitor sub command. It is strongly recommended that `--dry-run` be specified if the coordinator is invoked with the Akamai registrar.

An example command line invocation of the application utilizing the Akamai registrar is:

```
./edgedns-registrar-coordinator monitor --registrar akamai --interval 5m --edgegrid-edgerc-path /home/testuser/.edgerc --edgedns-contract 1-ABC123 --edgedns-group 99999 --dry-run --once
```

An example Akamai registrar configuration would be:

```
akamai_contracts: 1-CBA321

akamai_name_filter: edgedns

#akamai_edgerc_path: /home/test/.edgerc

#akamai_edgerc_section: registrar

akamai_host: akab-akamairegistrarh-ostexample123456.luna.akamaiapis.net
  
akamai_access_token: akab-akamaiaccesstoke-nexample12345678
  
akamai_client_token: akab-akamaiclienttoke-nexample12345678

akamai_client_secret: akamaiclientsecretexampleforregistrar123456=
```

### Plugin Registrar

The plugin registrar allows the coordinator to utilize additional registrars which are loaded dynamically at runtime. The plugin registrar utilizes the go [plugin](https://golang.org/pkg/plugin/) package. An Akamai plugin library is provided as an example for the development of other plugin registrars. The development of plugins is described further later in this document.

Note: Go plugins have limited platform support.

### Mark Monitor SFTP Registrar

The Mark Monitor sftp registrar enables the synchronization of Mark Monitor and Edge DNS.

An example command line invoking the application with the Mark Monitor registrar:

```
./edgedns-registrar-coordinator monitor --registrar markmonitorsftp  --interval 5m --dry-run --edgegrid-edgerc-path /home/testuser/.edgerc --edgedns-contract 1-ABC123 --edgedns-group 99999 --once --registrar-config-path /home/github.com/akamai/edgedns-registrar-coordinator/markmonitor-sftp-registrar-config-example.yaml

```

An example configuration file looks like:

```
# Mark Monitor SFTP Example Configuration
#
markmonitor_ssh_user: test

markmonitor_ssh_password: test

markmonitor_ssh_host: sftp_test_host

markmonitor_ssh_port: 22

# markmonitor_ssl_cert_algorithm:

# markmonitor_ssl_signature:

# markmonitor_sftp_pkt_size:

markmonitor_master_ips:
  - 1.2.3.4
  - 5.6.7.8

markmonitor_registrar_domain_filepath: testdomainconfig

markmonitor_temp_file_folder: /tmp
```

### Extending Coordinator Registrar Support

#### Integrated Registrars

The creation of an integrated registrar requires the development of the registrar itself as well as minor modification to the main.go source. 

The registrar must be located in a self identifying folder within the registrar source folder. The registrar must have a package name representative of its name. Finally, the registrar must implement the RegistrarProvider interface:

```
type RegistrarProvider interface {
        GetDomains(ctx context.Context) ([]string, error)
        GetDomain(ctx context.Context, domain string) (*Domain, error)
        GetTsigKey(ctx context.Context, domain string) (*dns.TSIGKey, error)
        GetServeAlgorithm(ctx context.Context, domain string) (string, error)
        GetMasterIPs(ctx context.Context) ([]string, error)
}
```

Registrars may define their own initialization function. However, it must return a registrar object and error. As an example, the plugin registrar initialization function is:

```
`func NewPluginRegistrar(ctx context.Context, pluginConfig registrar.PluginConfig) (*PluginRegistrar, error)`
``` 

The context passed to all functions handles abnormal termination as well as provides a logger.

Registrars must handle their own configuration. The `registrar-config-path` argument should be used to specify a registrar configuration. The new registrar might also reference environment variables or hard coded paths to additional configuration informtion. 

Lastly, the `main.go` source must be updated to import the new registrar package and invoke the new registrar's initialization function.

#### Plugin Library Registrars

New registrars may also be implemented as a go plugin library. The new registrar plugin library must implement the PluginProviderInterface:

```
type PluginRegistrarProvider interface {
	NewPluginLibRegistrar()
	GetDomains()
	GetDomain()
	GetTsigKey()
	GetServeAlgorithm()
	GetMasterIPs()
}
```

as well as expose variables to pass function args, result and error. The variable declarations must be as follows:

```
var(
	LibPluginArgs      registrar.PluginFuncArgs
	LibPluginResult    registrar.PluginFuncResult
}
```

PluginFuncArgs and registrar.PluginFuncResult are defined as the following structs:

```
type PluginFuncArgs struct {
	PluginArg interface{}
}

type PluginFuncResult struct {
	PluginResult interface{}
	PluginError  error
}
``` 

The plugin is passed the following configuration object when NewPluginLibRegistrar() is called:

```
type PluginConfig struct {
	PluginLibPath    string
	PluginName       string
	PluginConfigPath string
	LogEntry         *log.Entry
	Registrar        *plugin.Plugin
}
```

