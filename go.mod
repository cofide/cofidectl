module github.com/cofide/cofidectl

go 1.22.6

require (
	github.com/cofide/cofide-api-sdk v0.0.0-unpublished
	github.com/hashicorp/go-hclog v0.14.1
	github.com/hashicorp/go-plugin v1.6.1
	github.com/olekukonko/tablewriter v0.0.5
	github.com/spf13/cobra v1.8.1
	google.golang.org/grpc v1.67.1
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/cofide/cofide-connect v0.0.0-unpublished => ../cofide-connect

replace github.com/cofide/cofide-api-sdk v0.0.0-unpublished => ../cofide-api-sdk

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/color v1.7.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mitchellh/go-testing-interface v0.0.0-20171004221916-a61a99592b77 // indirect
	github.com/oklog/run v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240814211410-ddb44dafa142 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)
