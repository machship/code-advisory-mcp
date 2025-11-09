# Maskfile for code-advisory-mcp

## prepare

> Configure your environment with specific tools used by this project (run
> once).

```bash
git config --local include.path ../.gitconfig
cp -n .env.example .env
brew install mask
brew install mprocs
```

## build

> Tasks to build code-advisory-mcp binaries and other artifacts.

### build bin

> Build code-advisory-mcp service

```bash
go build -ldflags "-X 'main.version=$(git describe --always --tags --dirty="-$USER")'" -o bin/code-advisory-mcp .
```

### build image

> Build Docker image

```bash
docker build -f Dockerfile -t code-advisory-mcp .
```

### build db [manifest]

> Build build and export database.

```bash
go run ./cmd/builder -manifest ${manifest:-manifest.json} -backend ollama -model nomic-embed-text --output advisory.db
go run ./cmd/export -input advisory.db -output cmd/server/embedded/advisory.gob
```

## test

> Run all tests

```bash
go tool gotestsum --format pkgname -- ./...
```

## check

> Run formatter and checks, mostly static analysis.

Consider installing this in your pre-commit git hook.

```bash
go tool goimports -w .
mprocs --names staticcheck,errcheck,govulncheck,build \
    'go tool staticcheck ./...' \
    'go tool errcheck ./...' \
    'go tool govulncheck ./...' \
    'go build ./...'
```

## run

> Tasks to directly run code-advisory-mcp services.

### run service

> Run the service locally

```bash
go run .
```

## clean

> Clean build artifacts

```bash
rm -rf bin/
go clean -cache
```

## install

> Install/update dependencies

```bash
go mod tidy
go mod download
```

## generate

> Run go generate

```bash
go generate ./...
```
