module github.com/akamai/edgedns-registrar-coordinator/registrar/akamai-plugin-lib

go 1.14

require (
	github.com/akamai/AkamaiOPEN-edgegrid-golang v1.1.0
	github.com/akamai/edgedns-registrar-coordinator v0.0.0
	github.com/apex/log v1.9.0
	gopkg.in/yaml.v2 v2.4.0
)

replace (
        github.com/akamai/edgedns-registrar-coordinator => /home/github.com/akamai/edgedns-registrar-coordinator
)

