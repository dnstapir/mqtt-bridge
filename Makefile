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

tarball: outdir
	git archive --format=tar.gz --prefix=$(NAME)-$(VERSION)/ -o $(OUT)/$(NAME)-$(VERSION).tar.gz HEAD

container: build
	-rm -rf $(DOCKER_BUILDDIR)
	mkdir -p $(DOCKER_BUILDDIR)
	cp docker/Dockerfile $(DOCKER_BUILDDIR)
	cp docker/entrypoint.sh $(DOCKER_BUILDDIR)
	cp $(OUT)/$(NAME) $(DOCKER_BUILDDIR)
	cd $(DOCKER_BUILDDIR) && docker build . -t $(DOCKER_ITEST_IMAGE)

itest: container
	docker compose -f itests/sut/docker-compose.yaml -p mqtt-bridge-itest up -d
	go test ./itests
	docker compose -f itests/sut/docker-compose.yaml -p mqtt-bridge-itest stop
