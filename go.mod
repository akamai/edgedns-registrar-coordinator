module github.com/akamai/edgedns-registrar-coordinator

go 1.14

require (
	github.com/akamai/AkamaiOPEN-edgegrid-golang v1.1.0
	github.com/apex/log v1.9.0
	github.com/prometheus/common v0.18.0
	github.com/sirupsen/logrus v1.6.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.4.0
)

replace (
	github.com/akamai/edgedns-registrar-coordinator => /home/github.com/akamai/edgedns-registrar-coordinator
	github.com/akamai/edgedns-registrar-coordinator/internal => /home/github.com/akamai/edgedns-registrar-coordinator/internal
	github.com/akamai/edgedns-registrar-coordinator/registrar => /home/github.com/akamai/edgedns-registrar-coordinator/registrar
)
