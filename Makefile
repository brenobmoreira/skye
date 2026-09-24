TAGS := desktop,production,webkit2_41

.PHONY: frontend build test windows windows-icon install-windows windows-shortcuts

frontend:
	cd frontend && npm ci && npm run build

build: frontend
	go build -tags $(TAGS) -o skye ./cmd/skye

windows: frontend
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags desktop,production -ldflags "-H windowsgui" -o skye.exe ./cmd/skye-win

# Regenerates the icon resource go build embeds in skye.exe (id 3 is the one Wails loads) from
# cmd/skye-win/winres/icon.png; the .syso is committed, so a plain build does not need this.
windows-icon:
	cd cmd/skye-win && go run github.com/tc-hib/go-winres@v0.3.3 make --in winres/winres.json --out rsrc --arch amd64

install-windows: windows
	@appdata="$$(cmd.exe /c 'echo %LOCALAPPDATA%' 2>/dev/null | tr -d '\r')"; \
	test -n "$$appdata" || { echo "não achei o %LOCALAPPDATA% do Windows" >&2; exit 1; }; \
	dir="$$(wslpath "$$appdata")/skye"; \
	mkdir -p "$$dir" && cp skye.exe "$$dir/skye.exe" && \
	echo "instalado em $$(wslpath -w "$$dir/skye.exe")"
	@$(MAKE) --no-print-directory windows-shortcuts

# Points skye.lnk on the desktop and in the Start menu at the installed skye.exe. The shell
# folders are asked from Windows, so a desktop moved into OneDrive still gets the icon.
windows-shortcuts:
	@appdata="$$(cmd.exe /c 'echo %LOCALAPPDATA%' 2>/dev/null | tr -d '\r')"; \
	exe="$$appdata\\skye\\skye.exe"; \
	test -f "$$(wslpath "$$exe")" || { echo "skye.exe não instalado; rode make install-windows" >&2; exit 1; }; \
	powershell.exe -NoProfile -NonInteractive -Command "\
		\$$exe = '$$exe'; \$$ws = New-Object -ComObject WScript.Shell; \
		foreach (\$$f in 'Desktop', 'Programs') { \
			\$$l = \$$ws.CreateShortcut((Join-Path ([Environment]::GetFolderPath(\$$f)) 'skye.lnk')); \
			\$$l.TargetPath = \$$exe; \$$l.WorkingDirectory = (Split-Path \$$exe); \$$l.IconLocation = \"\$$exe,0\"; \
			\$$l.Save(); 'atalho em ' + \$$l.FullName }" | tr -d '\r'

test:
	go test ./internal/...
	cd frontend && npm test
