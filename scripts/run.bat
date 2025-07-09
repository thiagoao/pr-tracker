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

REM Clean unused imports and format code before running
echo Running goimports...
goimports -w .
echo Running go fmt...
go fmt ./...

echo Running Go app...
go run ./cmd 