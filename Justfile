bin := "cofidectl"
pkg := "./cmd/cofidectl/main.go"

# Internal build target without defaults
_build *args:
    CGO_ENABLED=0 go build {{args}} {{pkg}}

# Build without testing
build-only *args:
    just _build -o {{bin}} {{args}}

# Test and build
build *args: test
    just build-only {{args}}

# Release build with version injection
build-release-version version output=bin:
    just _build '-ldflags="-s -w -X main.version={{version}}"' -o {{output}}

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
