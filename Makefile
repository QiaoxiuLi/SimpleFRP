.PHONY: build test package-linux package-windows package-macos release

build:
	go build -trimpath -ldflags="-s -w" -o dist/simplefrp ./cmd/simplefrp

test:
	go test ./...

package-linux:
	./scripts/package-linux.sh

package-windows:
	pwsh ./scripts/package-windows.ps1

package-macos:
	./scripts/package-macos.sh

release:
	./scripts/release.sh
