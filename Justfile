build: test
    go build -o cofidectl ./cmd/cofidectl/main.go

test:
    go run gotest.tools/gotestsum@latest --format github-actions ./...

