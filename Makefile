TAGS := desktop,production,webkit2_41

.PHONY: frontend build test windows install-windows

frontend:
	cd frontend && npm ci && npm run build

build: frontend
	go build -tags $(TAGS) -o skye ./cmd/skye

windows: frontend
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags desktop,production -ldflags "-H windowsgui" -o skye.exe ./cmd/skye-win

install-windows:
	@appdata="$$(cmd.exe /c 'echo %LOCALAPPDATA%' 2>/dev/null | tr -d '\r')"; \
	test -n "$$appdata" || { echo "não achei o %LOCALAPPDATA% do Windows" >&2; exit 1; }; \
	dir="$$(wslpath "$$appdata")/skye"; \
	mkdir -p "$$dir" && cp skye.exe "$$dir/skye.exe" && \
	echo "instalado em $$(wslpath -w "$$dir/skye.exe")"

test:
	go test ./internal/...
	cd frontend && npm test
