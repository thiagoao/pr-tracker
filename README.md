# PR Tracker

A Pull Request monitoring service for Bitbucket that notifies about stale PRs through email and Microsoft Teams.

## 📋 Description

FC PR Tracker is a Go service that monitors Pull Requests in Bitbucket repositories and sends notifications when PRs become inactive for a configurable period. The service supports:

- Multiple repository monitoring
- Keyword filters to ignore specific PRs
- Email notifications (SMTP)
- Microsoft Teams notifications (webhook)
- Configurable logs with rotation
- Continuous execution with configurable intervals

## 🚀 Features

- **Automatic Monitoring**: Checks open PRs at configurable intervals
- **Smart Filters**: Ignores PRs with specific keywords (e.g., [WIP], [DRAFT])
- **Inactivity Detection**: Identifies PRs without activity for X days
- **Multiple Notifications**: Email and Microsoft Teams support
- **Structured Logs**: Configurable logging system with rotation
- **Flexible Configuration**: YAML file for all configurations

## 📦 Prerequisites

- Go 1.24.2 or higher
- Bitbucket API access (App Password)
- SMTP server (for email notifications)
- Microsoft Teams webhook (optional)

## 🛠️ Installation

### 1. Clone the repository
```bash
git clone <your-repository>
cd fc-pr-tracker
```

### 2. Install dependencies
```bash
go mod tidy
```

### 3. Configure the configuration file
```bash
cp config-example.yaml config.yaml
```

Edit the `config.yaml` file with your settings (see configuration section below).

## ⚙️ Configuration

### config.yaml file

Copy the `config-example.yaml` file to `config.yaml` and configure:

```yaml
log:
  file: "./logs/pr-tracker.log"
  level: "info"
  format: "text" # ou "json"
  max_size_mb: 10
  max_backups: 5
  max_age_days: 30
  compress: true
  stdout: false

bitbucket:
  domain: bitbucket.yourdomain.com
  port: 443
  workspace: your_workspace
  user: your_user
  app_password: your_app_password
  repositories:
    - your_repository1
    - your_repository2

pr_filter:
  ignore_keywords:
    - "[WIP]"
    - "[DRAFT]"
    - "[DO NOT MERGE]"
  stale_after_days: 3

notification:
  interval_hours: 12

notifiers:
  smtp:
    host: smtp.yourprovider.com
    port: 587
    user: your_email@domain.com
    password: your_password
    from: your_email@domain.com
    to:
      - recipient1@domain.com
      - recipient2@domain.com
  teams:
    webhook_url: "https://outlook.office.com/webhook/your-webhook-url"
```

### Bitbucket Settings

1. **App Password**: Create an App Password in Bitbucket with read permissions
2. **Workspace**: Workspace/organization name
3. **Repositories**: List of repositories to monitor

### Notification Settings

- **SMTP**: Configure your SMTP server for email sending
- **Teams**: Microsoft Teams webhook URL (optional)

## 🏗️ Build

### Windows
```bash
scripts/build-windows.bat
```
The executable will be created at: `bin/pr-tracker-windows.exe`

### Linux
```bash
scripts/build-linux.bat
```
The executable will be created at: `bin/pr-tracker-linux`

### Build Manual
```bash
# Windows
go build -o bin/pr-tracker.exe

# Linux
GOOS=linux GOARCH=amd64 go build -o bin/pr-tracker
```

## 🚀 Execution

### Development
```bash
# Windows (PowerShell)
scripts/run.bat

# Windows (PowerShell with GO111MODULE=on)
.\scripts\go-run.ps1 go run ./cmd

# Linux/Mac
make run
```

### Production
```bash
# Windows
bin/pr-tracker-windows.exe

# Linux
./bin/pr-tracker-linux
```

### Manual Execution
```bash
# Always enable Go modules
$env:GO111MODULE="on"; go run ./cmd

# Or use the helper script
.\scripts\go-run.ps1 go run ./cmd
```

## 📊 Logs

Logs are saved to `./logs/pr-tracker.log` by default. The system supports:

- Automatic log rotation
- Compression of old files
- Multiple formats (text/json)
- Logs to stdout (configurable)

## 🔧 Development

### Project Structure
```
fc-pr-tracker/
├── cmd/main.go          # Application entry point
├── internal/
│   ├── bitbucket/       # Bitbucket API client
│   ├── config/          # Configuration and YAML loading
│   ├── notifier/        # Notification implementations
│   └── logger/          # Logging configuration
├── pkg/models/          # Data models
├── config.yaml          # Application configuration
├── config-example.yaml   # Configuration example
├── scripts/             # Build and execution scripts
├── logs/                # Application logs
├── tmp/                 # Temporary files (state, etc.)
├── bin/                 # Compiled executables
└── docs/                # Documentation
```

### Testing

The project includes comprehensive tests for all components:

#### Running Tests
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with race detection
make test-race

# Run tests manually
go test -v ./pkg/... ./internal/... ./cmd/...
```

#### Test Structure
- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test component interactions
- **Coverage Reports**: HTML coverage reports generated automatically

#### Test Files
- `pkg/models/pull_request_test.go` - Tests for data models
- `internal/config/config_test.go` - Tests for configuration loading
- `internal/bitbucket/client_test.go` - Tests for Bitbucket client
- `internal/notifier/email_test.go` - Tests for email notifications
- `internal/notifier/teams_test.go` - Tests for Teams notifications
- `internal/logger/logger_test.go` - Tests for logging functionality
- `cmd/main_test.go` - Tests for main application logic

#### Test Coverage
The tests cover:
- Configuration loading and validation
- Bitbucket API client functionality
- Email and Teams notification generation
- Logging system initialization
- Data model operations
- File-based state persistence
- Error handling scenarios

### Useful Commands

```bash
# Format code
go fmt ./...

# Check imports
goimports -w .

# Run tests
go test ./...

# Clean dependencies
go mod tidy
```

## 🔍 Monitoring

The service runs in a continuous loop:

1. **Connection Check**: Tests Bitbucket connection on startup
2. **PR Monitoring**: Fetches open PRs from all configured repositories
3. **Filters**: Applies keyword filters and checks approvals
4. **Activity Analysis**: Calculates days without activity
5. **Notifications**: Sends notifications for stale PRs
6. **Interval**: Waits for the configured interval before the next check

## 🛡️ Error Handling

- Bitbucket connection tested on startup
- Detailed logs for debugging
- Graceful shutdown with Ctrl+C
- API error handling
- Notification fallback

## 📝 Example Logs

```
INFO PR monitoring script started log_file=./logs/pr-tracker.log log_level=info
INFO Bitbucket connection test succeeded
INFO Loaded configuration workspace=my-workspace user=my-user repositories=[repo1,repo2] stale_after_days=3
INFO Fetching open PRs for repository repo=my-repo
INFO Total open PRs repo=my-repo total=5
INFO PRs after keyword filter repo=my-repo filtered_total=3
INFO Sending summary notification email prs_to_notify=2
INFO Sleeping until next check... hours=12
```

## 🤝 Contributing

1. Fork the project
2. Create a branch for your feature (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

This project is under the MIT license. See the `LICENSE` file for more details.

## 🆘 Support

For support and questions:

1. Check logs at `./logs/pr-tracker.log`
2. Confirm settings in `config.yaml`
3. Test Bitbucket connection
4. Verify SMTP/Teams settings

## 🔄 Versions

- **v1.0.0**: Initial version with email and Teams support
- **Go 1.24.2**: Minimum Go version required 