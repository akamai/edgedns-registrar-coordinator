module github.com/akamai/edgedns-registrar-coordinator

go 1.14

require (
	github.com/akamai/AkamaiOPEN-edgegrid-golang v1.1.0
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/apex/log v1.9.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.6.0 // indirect
	github.com/stretchr/testify v1.7.0
	golang.org/x/sys v0.0.0-20200625212154-ddb9806d33ae // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.4.0
)

replace (
	github.com/akamai/edgedns-registrar-coordinator => /home/github.com/akamai/edgedns-registrar-coordinator
	github.com/akamai/edgedns-registrar-coordinator/internal => /home/github.com/akamai/edgedns-registrar-coordinator/internal
	github.com/akamai/edgedns-registrar-coordinator/registrar => /home/github.com/akamai/edgedns-registrar-coordinator/registrar
)
