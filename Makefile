.PHONY: all build install test coverage fmt vet clean tarball itest

NAME:=mqtt-bridge
OUT:=./out
DEFAULT_INSTALLDIR:=/usr/bin
INSTALL:=install -p -m 0755
VERSION:=0.0.0# Source of truth
COMMIT:=$$(git describe --dirty=+WiP --always 2> /dev/null || true)
DOCKER_BUILDDIR:=$(OUT)/dockertmp
DOCKER_ITEST_IMAGE:=mqtt-bridge-itest

all: build

build: outdir
	go build -v -ldflags "-X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)'" -o $(OUT)/ ./cmd/...

install:
	test "$(installdir)"    || $(INSTALL) $(OUT)/$(NAME) $(DEFAULT_INSTALLDIR)
	test -z "$(installdir)" || $(INSTALL) $(OUT)/$(NAME) $(installdir)

outdir:
	-mkdir -p $(OUT)

test:
	go test ./...

coverage: outdir
	go test -coverprofile=$(OUT)/coverage.out ./...
	go tool cover -html="$(OUT)/coverage.out" -o $(OUT)/coverage.html

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	-rm -rf $(OUT)
	-go clean -testcache

tarball: outdir
	git archive --format=tar.gz --prefix=$(NAME)-$(VERSION)/ -o $(OUT)/$(NAME)-$(VERSION).tar.gz HEAD

container: build
	cp ./itests/sut/Dockerfile ./out/Dockerfile
	docker build --tag mqtt-bridge:itest -f out/Dockerfile out/

itest: container
	go test -count=1 -v --tags=itests ./itests/...

bench: container
	go test -v -bench=. --tags=itests -count 5 -run=^\# ./itests
