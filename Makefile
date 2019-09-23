NAME := dpb
VERSION := $(shell git describe --tags --exact-match 2>/dev/null || \
			 printf %s\\n "git-$$(git describe --always --dirty)")

.PHONY: build
build: clean $(NAME)

# a temporary version file is created from a template in order to inject the VERSION variable into the static build
# embedding the version into the binary's DWARF table doesn't work because its stripped during the release build
TMP_VERSION_FILE := $(shell tr -dc 'a-f0-9' < /dev/urandom | dd bs=1 count=64 2>/dev/null).go
$(NAME): cmd/main.go
	sed 's/MAKE_VERSION/$(VERSION)/' .version > $(TMP_VERSION_FILE)
	cd cmd && go build -o build/$(NAME) .
	cd - && rm $(TMP_VERSION_FILE)

.PHONY: clean
clean:
	rm -f $(NAME)
	rm -rf build release


# release static crossbuilds

# CGO_ENABLED=0 and netgo not necessary; dpb doesn't call out to C or use net.
GO_LDFLAGS_STATIC=-ldflags "-s -w -extldflags -static"

define buildrelease
GOOS=$(1) GOARCH=$(2) go build -a \
	 -o release/$(NAME)-$(1)-$(2) \
	 $(GO_LDFLAGS_STATIC) . ;
sha512sum release/$(NAME)-$(1)-$(2) > release/$(NAME)-$(1)-$(2).sha512sum;
endef

GOOSARCHES = linux/arm linux/arm64 linux/amd64 darwin/amd64 openbsd/amd64 freebsd/amd64 netbsd/amd64

.PHONY: release
release: *.go
	$(foreach GOOSARCH,$(GOOSARCHES), $(call buildrelease,$(subst /,,$(dir $(GOOSARCH))),$(notdir $(GOOSARCH))))
