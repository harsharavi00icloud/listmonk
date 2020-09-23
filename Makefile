LAST_COMMIT := $(shell git rev-parse --short HEAD)
LAST_COMMIT_DATE := $(shell git show -s --format=%ci ${LAST_COMMIT})
VERSION := $(shell git describe)
BUILDSTR := ${VERSION} (${LAST_COMMIT} $(shell date -u +"%Y-%m-%dT%H:%M:%S%z"))

BIN := listmonk
STATIC := config.toml.sample \
	schema.sql queries.sql \
	static/public:/public \
	static/email-templates \
	frontend/dist/favicon.png:/frontend/favicon.png \
	frontend/dist/frontend:/frontend

# Dependencies.
.PHONY: deps
deps:
	go get -u github.com/knadh/stuffbin/...
	cd frontend && yarn install

# Build steps.
.PHONY: build
build:
	go build -o ${BIN} -ldflags="-s -w -X 'main.buildString=${BUILDSTR}' -X 'main.versionString=${VERSION}'" ./cmd/listmonk

.PHONY: build-frontend
build-frontend:
	export VUE_APP_VERSION="${VERSION}" && cd frontend && yarn build

.PHONY: run
run: 
	go run ./cmd/listmonk	

.PHONY: run-frontend
run-frontend:
	export VUE_APP_VERSION="${VERSION}" && cd frontend && yarn serve

.PHONY: test
test:
	go test ./...

# dist builds the backend, frontend, and uses stuffbin to
# embed all frontend assets into the binary.
.PHONY: dist
dist: build build-frontend
	stuffbin -a stuff -in ${BIN} -out ${BIN} ${STATIC}

# pack-releases runns stuffbin packing on a given list of
# binaries. This is used with goreleaser for packing
# release builds for cross-build targets.
.PHONY: pack-releases
pack-releases:
	$(foreach var,$(RELEASE_BUILDS),stuffbin -a stuff -in ${var} -out ${var} ${STATIC};)

