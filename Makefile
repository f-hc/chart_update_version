# SPDX-License-Identifier: GPL-3.0-only
#
# Copyright (C) 2026 f-hc <207619282+f-hc@users.noreply.github.com>
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, version 3 of the License.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

BINARY := updater
GO := go

.PHONY: all build clean test lint fmt vet run check help

all: build-all

build: ## Build binary for current platform
	$(GO) build -ldflags '-s -w' -trimpath -o $(BINARY) .

clean: ## Remove built binaries
	rm -f $(BINARY) $(BINARY)-*

build-all: build-darwin build-linux build-freebsd ## Build binaries for all platforms

build-darwin: ## Build binaries for macOS (amd64, arm64)
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags '-s -w' -trimpath -o $(BINARY)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags '-s -w' -trimpath -o $(BINARY)-darwin-arm64 .

build-linux: ## Build binaries for Linux (amd64, arm64)
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags '-s -w' -trimpath -o $(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GO) build -ldflags '-s -w' -trimpath -o $(BINARY)-linux-arm64 .

build-freebsd: ## Build binaries for FreeBSD (amd64, arm64)
	GOOS=freebsd GOARCH=amd64 $(GO) build -ldflags '-s -w' -trimpath -o $(BINARY)-freebsd-amd64 .
	GOOS=freebsd GOARCH=arm64 $(GO) build -ldflags '-s -w' -trimpath -o $(BINARY)-freebsd-arm64 .

test: ## Run tests
	$(GO) test -v ./...

lint: ## Run golangci-lint
	golangci-lint run

fmt: ## Format code with go fmt
	$(GO) fmt ./...

vet: ## Run go vet
	$(GO) vet ./...

run: build ## Build and run the binary
	./$(BINARY)

check: build ## Build and run with -C flag (discover charts)
	./$(BINARY) -C

help: ## Show this help message
	@awk -F ':.*?## ' '/^[a-zA-Z_-]+:.*## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
