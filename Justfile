bin := "cofidectl"
pkg := "./cmd/cofidectl/main.go"

# Build without testing
build-only *args:
    CGO_ENABLED=0 go build -o {{bin}} {{args}} {{pkg}}

# Test and build
build *args: test
    just build-only {{args}}

# Release build with version injection
release-version output=bin version:
    CGO_ENABLED=0 go build -ldflags="-s -w -X main.version={{version}}" -o {{output}} {{pkg}}

# Build test plugin without testing
build-test-plugin-only:
    CGO_ENABLED=0 go build -o cofidectl-test-plugin ./cmd/cofidectl-test-plugin/main.go

# Test and build plugin
build-test-plugin: test build-test-plugin-only

# Release build for test plugin
build-test-plugin-release output="cofidectl-test-plugin":
    CGO_ENABLED=0 go build -ldflags="-s -w" -o {{output}} ./cmd/cofidectl-test-plugin/main.go

install-test-plugin: build-test-plugin
    mkdir -p ~/.cofide/plugins
    cp cofidectl-test-plugin ~/.cofide/plugins

test *args:
    CGO_ENABLED=0 go run gotest.tools/gotestsum@latest --format github-actions ./... {{args}}


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
