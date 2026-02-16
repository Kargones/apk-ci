# apk-ci Project Context

## Project Overview

apk-ci is a Go-based automation tool for working with 1C:Enterprise systems. It provides modules for:

- Converting data between formats
- Restoring and managing MSSQL databases
- Managing service mode for 1C information bases
- Integrating with 1C:Enterprise Development Tools (EDT)
- Centralized configuration management

The project is built with a modular architecture where each module handles a specific area of 1C:Enterprise functionality.

## Key Technologies

- **Language**: Go (1.25.0)
- **Key Libraries**: 
  - `github.com/denisenkom/go-mssqldb` (MSSQL connectivity)
  - `github.com/ilyakaznacheev/cleanenv` (environment configuration parsing)
  - `gopkg.in/yaml.v3` (YAML processing)
- **Testing**: `github.com/stretchr/testify`

## Project Structure

```
apk-ci/
├── cmd/
│   └── apk-ci/
│       └── main.go              # Entry point
├── internal/
│   ├── app/                     # Main application logic
│   ├── config/                  # Configuration management
│   ├── constants/               # Project constants
│   ├── servicemode/             # Service mode management
│   ├── rac/                     # RAC (Remote Administration Console) client
│   ├── entity/                  # Module-specific implementations
│   │   ├── dbrestore/           # Database restore functionality
│   │   ├── one/                 # 1C platform integration
│   │   └── ...                  # Other modules
│   └── util/                    # Utility functions
├── docs/                        # Technical documentation
├── .wiki/                       # User documentation
├── Makefile                     # Build and test commands
├── go.mod                       # Go module dependencies
└── README.md                    # Project overview
```

## Key Modules

### Service Mode Management

The service mode module allows enabling/disabling service mode for 1C information bases, which blocks user access for administrative operations.

**Key files**:
- `internal/servicemode/servicemode.go` - Main service mode management interface
- `internal/rac/service_mode.go` - RAC client implementation for service mode operations
- `internal/rac/rac.go` - Generic RAC client functionality

**Commands**:
- `service-mode-enable` - Enable service mode for an information base
- `service-mode-disable` - Disable service mode for an information base
- `service-mode-status` - Check service mode status for an information base

### Configuration Management

The project uses a centralized configuration system that supports:
1. Environment variables (highest priority)
2. Configuration files (YAML/JSON)
3. Default values (lowest priority)

**Key files**:
- `internal/config/config.go` - Main configuration loading and management
- `internal/constants/constants.go` - Project constants

### Database Operations

Supports restoring and managing MSSQL databases with automatic timeout calculation and error recovery mechanisms.

**Key files**:
- `internal/entity/dbrestore/` - Database restore functionality

### Git Integration

Integrates with Git/Gitea systems for development workflow automation.

## Building and Running

### Build Commands

```bash
# Build the application
make build

# Build for all platforms
make build-all

# Install the application
make install
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration
```

### Running the Application

```bash
# View help
./build/apk-ci --help

# Enable service mode
./build/apk-ci service-mode-enable --infobase MyInfobase

# Disable service mode
./build/apk-ci service-mode-disable --infobase MyInfobase

# Check service mode status
./build/apk-ci service-mode-status --infobase MyInfobase
```

## Configuration

### Environment Variables

The application uses environment variables for configuration with these prefixes:
- **ServiceMode**: `SERVICE_MODE_`
- **DBRestore**: `DBRESTORE_`
- **Convert**: `CONVERT_`
- **EDT**: `EDT_`

### Configuration Files

Configuration is loaded from YAML files:
- `app.yaml` - Application settings
- `project.yaml` - Project settings
- `secret.yaml` - Secrets (passwords, tokens)
- `dbconfig.yaml` - Database configuration

### Example Configuration

```yaml
# app.yaml
logLevel: "Debug"
workDir: "/tmp/benadis"
tmpDir: "/tmp/benadis/temp"
timeout: 30
paths:
  bin1cv8: "/opt/1cv8/x86_64/8.3.27.1606/1cv8"
  binIbcmd: "/opt/1cv8/x86_64/8.3.27.1606/ibcmd"
  edtCli: "/opt/1C/1CE/components/1c-edt-2024.2.6+7-x86_64/1cedtcli"
  rac: "/opt/1cv8/x86_64/8.3.27.1606/rac"
rac:
  port: 1545
  timeout: 30
  retries: 3
users:
  rac: "admin"
  db: "db_user"
  mssql: "mssql_user"
  storeAdmin: "store_admin"
```

## Development Conventions

### Code Organization

- Follows modular architecture with clear separation of concerns
- Uses dependency injection for components
- Implements adapter pattern for 1C platform version compatibility
- Uses strategy pattern for configuration sources
- Implements factory pattern for module-specific clients

### Error Handling

- Structured error logging with context information
- Automatic retries with exponential backoff for transient failures
- Graceful degradation when external systems fail
- Rollback mechanisms for critical operations

### Testing

- Comprehensive unit tests for business logic
- Mocking of external dependencies
- Integration tests for full scenarios
- Coverage analysis

### Security

- Separate storage of secrets
- Encryption of sensitive information
- Restricted file permissions for configuration files
- Audit logging of all critical operations

## Key Components

### Main Entry Point

`cmd/apk-ci/main.go` is the application entry point that:
1. Loads configuration
2. Routes to appropriate command handlers based on `BR_COMMAND` environment variable
3. Handles errors and exit codes

### Service Mode Implementation

The service mode functionality works through these layers:
1. **Main** (`main.go`) - Routes service mode commands
2. **App** (`internal/app/app.go`) - High-level service mode operations
3. **ServiceMode** (`internal/servicemode/servicemode.go`) - Service mode client management
4. **RAC** (`internal/rac/service_mode.go`) - Direct RAC command execution

### RAC Client

The RAC (Remote Administration Console) client in `internal/rac/` provides:
- Connection to 1C servers
- Command execution with retries
- Output parsing and error handling
- Specific service mode operations (enable/disable/status)

## Documentation

Documentation is available in:
- `README.md` - Project overview and basic usage
- `.wiki/` directory - Comprehensive user documentation
- `docs/` directory - Technical documentation and diagrams
- Source code comments - Detailed function and method documentation

## Common Operations

### Enabling Service Mode

```bash
# Enable service mode without terminating sessions
./build/apk-ci service-mode-enable --infobase MyInfobase

# Enable service mode and terminate active sessions
./build/apk-ci service-mode-enable --infobase MyInfobase --terminate-sessions
```

### Disabling Service Mode

```bash
# Disable service mode
./build/apk-ci service-mode-disable --infobase MyInfobase
```

### Checking Service Mode Status

```bash
# Check service mode status
./build/apk-ci service-mode-status --infobase MyInfobase
```

## Exit Codes

- `0` - Success
- `2` - Unknown command
- `5` - Configuration loading error
- `8` - Service mode operation error