module github.com/cofide/cofidectl

go 1.22.6

require (
	cuelang.org/go v0.10.0
	github.com/cofide/cofide-api-sdk v0.0.0-unpublished
	github.com/cofide/cofide-connect v0.0.0-unpublished
	github.com/hashicorp/go-hclog v0.14.1
	github.com/hashicorp/go-plugin v1.6.1
	github.com/olekukonko/tablewriter v0.0.5
	github.com/spf13/cobra v1.8.1
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56
	google.golang.org/grpc v1.67.1
	google.golang.org/protobuf v1.34.2
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/rogpeppe/go-internal v1.12.1-0.20240709150035-ccf4b4329d21 // indirect

replace github.com/cofide/cofide-connect v0.0.0-unpublished => ../cofide-connect

replace github.com/cofide/cofide-api-sdk v0.0.0-unpublished => ../cofide-api-sdk

require (
	cuelabs.dev/go/oci/ociregistry v0.0.0-20240807094312-a32ad29eed79 // indirect
	github.com/cockroachdb/apd/v3 v3.2.1 // indirect
	github.com/emicklei/proto v1.13.2 // indirect
	github.com/fatih/color v1.7.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mitchellh/go-testing-interface v0.0.0-20171004221916-a61a99592b77 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/oklog/run v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/protocolbuffers/txtpbfmt v0.0.0-20230328191034-3462fbc510c0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spiffe/spire-api-sdk v1.10.1 // indirect
	golang.org/x/mod v0.20.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/oauth2 v0.22.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240814211410-ddb44dafa142 // indirect
)
