.PHONY: run

# Golang Flags
GOFLAGS ?= $(GOFLAGS:)
GO=go

run:
	GO111MODULE=on $(GO) run $(GOFLAGS) $(GO_LINKER_FLAGS) *.go


vendor:
	@echo "Running $@"
	GO111MODULE=on go mod vendor