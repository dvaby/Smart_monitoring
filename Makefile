.PHONY: test frontend architecture build-linux build-windows

test:
	go test -tags "$(TAGS)" ./...

frontend:
	cd frontend && npm ci && npm run build

architecture:
	GOFLAGS="-tags=$(TAGS)" go run github.com/OpenPeeDeeP/depguard/v2/cmd/depguard@v2.3.0 ./...

build-linux: frontend
	wails build -platform linux/amd64 -tags "$(TAGS)"

build-windows: frontend
	wails build -platform windows/amd64
