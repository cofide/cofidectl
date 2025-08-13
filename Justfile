build: build-only test

build-only:
    go build -o cofidectl ./cmd/cofidectl/main.go

build-test-plugin:
    go build -o cofidectl-test-plugin ./cmd/cofidectl-test-plugin/main.go

install-test-plugin: build-test-plugin
    mkdir -p ~/.cofide/plugins
    cp cofidectl-test-plugin ~/.cofide/plugins

test *args:
    go run gotest.tools/gotestsum@latest --format github-actions ./... {{args}}

test-race: (test "--" "-race")

lint *args:
    golangci-lint run {{args}}

install-kind:
    tests/integration/install-kind.sh

create-kind-cluster:
    tests/integration/create-kind-cluster.sh

create-kind-clusters num_clusters:
    tests/integration/create-kind-clusters.sh {{num_clusters}}

integration-test test:
    tests/integration/{{test}}/test.sh
