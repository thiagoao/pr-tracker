@echo off
REM Always enable Go modules
set GO111MODULE=on

REM Ensure goimports is installed
where goimports >nul 2>nul
IF %ERRORLEVEL% NEQ 0 (
    echo Installing goimports...
    go install golang.org/x/tools/cmd/goimports@latest
) ELSE (
    echo goimports already installed.
)

REM Run go mod tidy to clean up dependencies
echo Running go mod tidy...
go mod tidy

echo Running goimports...
goimports -w .
echo Running go fmt...
go fmt ./...

set GOOS=linux
set GOARCH=amd64
REM Always create bin at project root
if not exist "%~dp0..\bin" mkdir "%~dp0..\bin"
echo Building Linux executable at project root...
go build -o "%~dp0..\bin\pr-tracker-linux" ./cmd 