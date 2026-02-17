# Environment Variables

<cite>
**Referenced Files in This Document**   
- [config.go](file://internal/config/config.go)
- [sonarqube.go](file://internal/config/sonarqube.go)
- [action.yaml](file://config/action.yaml)
- [constants.go](file://internal/constants/constants.go)
- [providers.go](file://internal/di/providers.go)
- [factory.go](file://internal/pkg/output/factory.go)
- [factory.go](file://internal/pkg/logging/factory.go)
</cite>

## Update Summary
**Changes Made**   
- Added new logging environment variables (BR_LOG_LEVEL, BR_LOG_FORMAT, BR_LOG_OUTPUT)
- Added new output format environment variable (BR_OUTPUT_FORMAT)
- Updated logging defaults to use text format and stderr output
- Added output writer selection mechanism

## Table of Contents
1. [Configuration Mechanism Overview](#configuration-mechanism-overview)
2. [Environment Variable Binding with Struct Tags](#environment-variable-binding-with-struct-tags)
3. [Supported Environment Variables](#supported-environment-variables)
4. [Logging Configuration Variables](#logging-configuration-variables)
5. [Output Writer Configuration](#output-writer-configuration)
6. [Configuration Loading Priority](#configuration-loading-priority)
7. [Practical Examples for Different Workflows](#practical-examples-for-different-workflows)
8. [Common Issues and Troubleshooting](#common-issues-and-troubleshooting)
9. [Best Practices for CI/CD and Development](#best-practices-for-cicd-and-development)

## Configuration Mechanism Overview

The apk-ci application implements a comprehensive configuration system that prioritizes environment variables over YAML configuration files using the cleanenv library. This mechanism allows for flexible configuration management across different environments while maintaining backward compatibility with existing YAML-based configurations.

The configuration system follows a hierarchical approach where environment variables take precedence over file-based configurations, enabling dynamic runtime adjustments without modifying configuration files. The cleanenv library is used throughout the codebase to parse environment variables and bind them to struct fields, providing type safety and default value support.

The primary configuration structure is defined in the Config struct within internal/config/config.go, which serves as the central repository for all application settings. This struct integrates various configuration sources including environment variables, YAML files (app.yaml, project.yaml, secret.yaml, dbconfig.yaml), and command-line inputs from GitHub Actions.

**Section sources**
- [config.go](file://internal/config/config.go#L130-L200)

## Environment Variable Binding with Struct Tags

The apk-ci application uses struct tags to establish the binding between environment variables and configuration fields. Each field in the Config struct is annotated with `env` and `env-default` tags that specify the corresponding environment variable name and default value.

The cleanenv library reads these struct tags during configuration loading and automatically populates the struct fields with values from environment variables when available. If an environment variable is not set, the library uses the specified default value or leaves the field empty if no default is provided.

For example, the Config struct contains fields like:
- `Actor string `env:"BR_ACTOR" env-default:"""`` - binds to BR_ACTOR environment variable
- `Env string `env:"BR_ENV" env-default:"dev"`` - binds to BR_ENV with "dev" as default
- `Command string `env:"BR_COMMAND" env-default:""`` - binds to BR_COMMAND environment variable

This approach provides a clean separation between configuration definition and implementation, allowing developers to easily understand which environment variables affect specific configuration parameters.

**Section sources**
- [config.go](file://internal/config/config.go#L130-L150)

## Supported Environment Variables

The apk-ci supports a comprehensive set of environment variables that control various aspects of application behavior. These variables are categorized by their functional areas and have specific purposes, default values, and impacts on application execution.

### Core Application Variables
These variables control fundamental application behavior and are required for basic operation:

- **BR_ACTOR**: Specifies the user who initiated the process. No default value; must be provided.
- **BR_ENV**: Sets the environment context (e.g., dev, test, prod). Default: "dev".
- **BR_COMMAND**: Defines the command to execute. No default value; must be provided.
- **BR_INFOBASE_NAME**: Specifies the name of the information base for service mode operations. No default value.
- **BR_TERMINATE_SESSIONS**: Boolean flag to terminate active sessions when enabling service mode. Default: "false".
- **BR_FORCE_UPDATE**: Boolean flag to force operations even when not necessary. Default: "false".
- **BR_ISSUE_NUMBER**: Integer specifying the issue number in Gitea. Default: 1.
- **BR_START_EPF**: URL to an external processing file (.epf) for execution. No default value.
- **BR_EXT_DIR**: Directory for extensions. No default value.
- **BR_DRY_RUN**: Boolean flag to perform dry run operations. Default: "false".

### Configuration File Path Variables
These variables specify the locations of configuration files:

- **BR_CONFIG_SYSTEM**: Path to the system configuration file (app.yaml). No default value.
- **BR_CONFIG_PROJECT**: Path to the project configuration file (project.yaml). No default value.
- **BR_CONFIG_SECRET**: Path to the secret configuration file (secret.yaml). No default value.
- **BR_CONFIG_DBDATA**: Path to the database configuration file (dbconfig.yaml). No default value.
- **BR_CONFIG_MENU_MAIN**: Path to the main menu configuration file (menu_main.yaml). No default value.
- **BR_CONFIG_MENU_DEBUG**: Path to the debug menu configuration file (menu_debug.yaml). No default value.

### Implementation Selection Variables
These variables control which implementation tools to use:

- **BR_IMPL_CONFIG_EXPORT**: Tool for configuration export ("1cv8", "ibcmd", "native"). Default: "1cv8".
- **BR_IMPL_DB_CREATE**: Tool for database creation ("1cv8", "ibcmd"). Default: "1cv8".

### Git Configuration Variables
These variables configure Git operations:

- **GIT_USER_NAME**: Username for Git operations. Default: "apk-ci".
- **GIT_USER_EMAIL**: Email address for Git operations. Default: "runner@benadis.ru".
- **GIT_DEFAULT_BRANCH**: Default branch name. Default: "main".
- **GIT_TIMEOUT**: Timeout for Git operations. Default: 30 seconds.
- **GIT_CREDENTIAL_HELPER**: Credential helper configuration. Default: "store".
- **GIT_CREDENTIAL_TIMEOUT**: Timeout for credential cache. Default: 300 seconds.

### RAC Configuration Variables
These variables configure Remote Administration Console operations:

- **RAC_PATH**: Path to the RAC executable. Default: "/opt/1cv8/x86_64/8.3.25.1257/rac".
- **RAC_SERVER**: Address of the RAC server. Default: "localhost".
- **RAC_PORT**: Port number for RAC server. Default: 1545.
- **RAC_USER**: Username for RAC authentication. No default value.
- **RAC_PASSWORD**: Password for RAC authentication. No default value.
- **RAC_DB_USER**: Database user for RAC operations. No default value.
- **RAC_DB_PASSWORD**: Database password for RAC operations. No default value.
- **RAC_TIMEOUT**: Timeout for RAC operations. Default: 30 seconds.
- **RAC_RETRIES**: Number of retry attempts for RAC operations. Default: 3.

### Database Restore Configuration Variables
These variables configure database restore operations:

- **DBRESTORE_SERVER**: Database server for restore operations. No default value.
- **DBRESTORE_USER**: Database user for restore operations. No default value.
- **DBRESTORE_PASSWORD**: Database password for restore operations. No default value.
- **DBRESTORE_DATABASE**: Database name for restore operations. No default value.
- **DBRESTORE_BACKUP**: Path to backup file for restore operations. No default value.
- **DBRESTORE_TIMEOUT**: Timeout for restore operations. Default: 30 seconds.
- **DBRESTORE_SRC_SERVER**: Source server for restore operations. No default value.
- **DBRESTORE_SRC_DB**: Source database for restore operations. No default value.
- **DBRESTORE_DST_SERVER**: Destination server for restore operations. No default value.
- **DBRESTORE_DST_DB**: Destination database for restore operations. No default value.
- **DBRESTORE_AUTOTIMEOUT**: Boolean flag for automatic timeout calculation. Default: false.

### Service Mode Configuration Variables
These variables configure service mode operations:

- **SERVICE_RAC_PATH**: Path to RAC executable for service mode. No default value.
- **SERVICE_RAC_SERVER**: RAC server address for service mode. No default value.
- **SERVICE_RAC_PORT**: RAC port for service mode. No default value.
- **SERVICE_RAC_USER**: RAC user for service mode. No default value.
- **SERVICE_RAC_PASSWORD**: RAC password for service mode. No default value.
- **SERVICE_DB_USER**: Database user for service mode. No default value.
- **SERVICE_DB_PASSWORD**: Database password for service mode. No default value.
- **SERVICE_RAC_TIMEOUT**: Timeout for service mode operations. Default: 30 seconds.
- **SERVICE_RAC_RETRIES**: Retry attempts for service mode operations. Default: 3.

### EDT Configuration Variables
These variables configure Enterprise Development Tools:

- **EDT_CLI_PATH**: Path to EDT CLI executable. No default value.
- **EDT_WORKSPACE**: EDT workspace directory. No default value.
- **EDT_PROJECT_DIR**: EDT project directory. No default value.

### GitHub Actions Integration Variables
These variables are passed through GitHub Actions inputs:

- **INPUT_DBNAME**: Database name from GitHub Actions. No default value.
- **INPUT_CONFIGSECRET**: Secret configuration path from GitHub Actions. No default value.
- **INPUT_TERMINATESESSIONS**: Terminate sessions flag from GitHub Actions. No default value.
- **INPUT_ACTOR**: Actor from GitHub Actions. No default value.
- **INPUT_CONFIGPROJECT**: Project configuration path from GitHub Actions. No default value.
- **INPUT_COMMAND**: Command from GitHub Actions. No default value.
- **INPUT_ISSUENUMBER**: Issue number from GitHub Actions. No default value.
- **INPUT_LOGLEVEL**: Log level from GitHub Actions. No default value.
- **INPUT_CONFIGSYSTEM**: System configuration path from GitHub Actions. No default value.
- **INPUT_CONFIGDBDATA**: Database configuration path from GitHub Actions. No default value.
- **INPUT_ACCESSTOKEN**: Access token from GitHub Actions. No default value.
- **INPUT_GITEAURL**: Gitea URL from GitHub Actions. No default value.
- **INPUT_REPOSITORY**: Repository from GitHub Actions. No default value.
- **INPUT_FORCE_UPDATE**: Force update flag from GitHub Actions. No default value.
- **INPUT_MENUMAIN**: Main menu configuration from GitHub Actions. No default value.
- **INPUT_MENUDEBUG**: Debug menu configuration from GitHub Actions. No default value.
- **INPUT_STARTEPF**: Start EPF URL from GitHub Actions. No default value.
- **INPUT_BRANCHFORSCAN**: Branch for scan from GitHub Actions. No default value.
- **INPUT_COMMITHASH**: Commit hash from GitHub Actions. No default value.

**Section sources**
- [config.go](file://internal/config/config.go#L130-L200)
- [sonarqube.go](file://internal/config/sonarqube.go#L15-L50)

## Logging Configuration Variables

The apk-ci includes comprehensive logging configuration through environment variables with the `BR_LOG_*` prefix. These variables control the logging behavior and output format.

### Logging Level Configuration
- **BR_LOG_LEVEL**: Sets the logging verbosity level. Valid values: "debug", "info", "warn", "error". Default: "info".
- **INPUT_LOGLEVEL**: GitHub Actions input for log level override. Default: empty.

### Logging Format Configuration
- **BR_LOG_FORMAT**: Sets the log output format. Valid values: "json", "text". Default: "text".
- **BR_LOG_OUTPUT**: Sets the log output destination. Valid values: "stdout", "stderr", "file". Default: "stderr".

### File Logging Configuration
When BR_LOG_OUTPUT is set to "file", additional variables control file logging:

- **BR_LOG_FILE_PATH**: Path to log file when output is "file". Default: "/var/log/apk-ci.log".
- **BR_LOG_MAX_SIZE**: Maximum log file size in MB. Default: 100.
- **BR_LOG_MAX_BACKUPS**: Maximum number of backup log files. Default: 3.
- **BR_LOG_MAX_AGE**: Maximum age of backup log files in days. Default: 7.
- **BR_LOG_COMPRESS**: Whether to compress backup log files. Default: true.

**Updated** Added comprehensive logging configuration variables with new defaults for improved readability and separation of concerns.

**Section sources**
- [config.go](file://internal/config/config.go#L328-L353)
- [config.go](file://internal/config/config.go#L597-L609)

## Output Writer Configuration

The apk-ci includes a flexible output writer system controlled by the `BR_OUTPUT_FORMAT` environment variable. This system allows switching between different output formats without restarting the application.

### Output Format Selection
- **BR_OUTPUT_FORMAT**: Controls the output format for command results. Valid values: "json", "text". Default: "text".

### Output Writer Behavior
The output writer system provides:
- **JSON Format**: Structured JSON output suitable for machine processing
- **Text Format**: Human-readable text output suitable for console display
- **Automatic Selection**: Uses environment variable to determine output format
- **Default Fallback**: Falls back to text format if environment variable is empty or invalid

### Output Writer Factory
The output writer is created through a factory pattern:
- `output.NewWriter()` creates writers based on format string
- Supports "json" and "text" formats
- Returns TextWriter as default for unknown formats
- Independent of Config struct for flexibility

**Updated** Added new output writer configuration system with BR_OUTPUT_FORMAT environment variable for flexible output format selection.

**Section sources**
- [providers.go](file://internal/di/providers.go#L36-L51)
- [factory.go](file://internal/pkg/output/factory.go#L9-L22)

## Configuration Loading Priority

The apk-ci follows a specific priority order when loading configuration values, ensuring that more specific or runtime-defined values override general or static ones. The priority hierarchy from highest to lowest is:

1. **Environment Variables**: Highest priority, allowing runtime overrides
2. **GitHub Actions Inputs**: Passed through INPUT_* environment variables
3. **Secret Configuration (secret.yaml)**: Contains sensitive data like passwords and tokens
4. **Application Configuration (app.yaml)**: Contains system-wide settings
5. **Default Values**: Hardcoded defaults in the configuration structs

This priority system enables flexible configuration management where environment variables can override any setting defined in configuration files. For example, the SonarQube token can be specified in secret.yaml but overridden by the SONARQUBE_TOKEN environment variable at runtime.

The MustLoad function in config.go orchestrates this loading process by first reading environment variables into the InputParams struct, validating required parameters, and then progressively loading configuration from various sources while respecting the priority hierarchy.

When multiple configuration sources provide values for the same parameter, the higher-priority source takes precedence. This allows for environment-specific overrides without modifying configuration files, which is particularly useful in CI/CD pipelines where different stages may require different settings.

**Section sources**
- [config.go](file://internal/config/config.go#L200-L400)

## Practical Examples for Different Workflows

### Database Restore Workflow
To perform a database restore operation with enhanced logging:

```bash
export BR_COMMAND=dbrestore
export BR_INFOBASE_NAME="test_database"
export BR_ACTOR="gitops-bot"
export BR_LOG_LEVEL="debug"
export BR_LOG_FORMAT="json"
export BR_LOG_OUTPUT="stderr"
export BR_GITEAURL="https://gitea.example.com"
export BR_REPOSITORY="organization/project"
export BR_ACCESS_TOKEN="your-access-token"
export DBRESTORE_SERVER="localhost"
export DBRESTORE_DATABASE="test_database"
export DBRESTORE_BACKUP="/backups/test_database.bak"
export DBRESTORE_TIMEOUT="300s"
```

This configuration will initiate a database restore operation using the specified backup file, with detailed JSON logs sent to stderr.

### Service Mode Management
To enable service mode for maintenance operations with custom output:

```bash
export BR_COMMAND=service-mode-enable
export BR_INFOBASE_NAME="production_db"
export BR_ACTOR="admin-user"
export BR_OUTPUT_FORMAT="json"
export BR_GITEAURL="https://gitea.example.com"
export BR_REPOSITORY="organization/project"
export BR_ACCESS_TOKEN="your-access-token"
export BR_TERMINATE_SESSIONS="true"
```

To disable service mode after maintenance:

```bash
export BR_COMMAND=service-mode-disable
export BR_INFOBASE_NAME="production_db"
export BR_ACTOR="admin-user"
export BR_OUTPUT_FORMAT="text"
export BR_GITEAURL="https://gitea.example.com"
export BR_REPOSITORY="organization/project"
export BR_ACCESS_TOKEN="your-access-token"
```

### SonarQube Scanning
To scan a specific branch with SonarQube and custom logging:

```bash
export BR_COMMAND=sq-scan-branch
export BR_ACTOR="ci-bot"
export BR_OUTPUT_FORMAT="json"
export BR_LOG_LEVEL="info"
export BR_GITEAURL="https://gitea.example.com"
export BR_REPOSITORY="organization/project"
export BR_ACCESS_TOKEN="your-access-token"
export SONARQUBE_URL="https://sonarqube.example.com"
export SONARQUBE_TOKEN="your-sonarqube-token"
export INPUT_BRANCHFORSCAN="feature/new-module"
export INPUT_COMMITHASH="a1b2c3d4e5f6"
```

This configuration will trigger a SonarQube scan on the specified branch and commit, using JSON output format for machine processing.

**Section sources**
- [config.go](file://internal/config/config.go#L130-L200)
- [action.yaml](file://config/action.yaml#L1-L120)
- [constants.go](file://internal/constants/constants.go#L1-L200)

## Common Issues and Troubleshooting

### Incorrect Naming Conventions
One common issue is incorrect environment variable naming. Ensure that:
- All BR_* variables use uppercase letters and underscores
- Logging variables use the BR_LOG_* pattern (BR_LOG_LEVEL, BR_LOG_FORMAT, BR_LOG_OUTPUT)
- Output variables use the BR_OUTPUT_FORMAT pattern
- SonarQube variables follow the SONARQUBE_* pattern
- Git variables use the GIT_* pattern
- RAC variables use the RAC_* pattern

For example, use `BR_LOG_LEVEL` instead of `br_log_level` or `BrLogLevel`.

### Missing Required Variables
Certain variables are required for the application to function properly:
- `BR_ACTOR`: Must be specified to identify the initiating user
- `BR_COMMAND`: Must be specified to determine the action to perform
- `BR_GITEAURL`: Must be specified to connect to the Gitea server
- `BR_REPOSITORY`: Must be specified to identify the target repository
- `BR_ACCESS_TOKEN`: Must be specified for authentication

Missing any of these required variables will cause the application to fail during startup validation.

### Type Mismatches
Ensure that environment variables are set with appropriate types:
- Boolean values should be "true" or "false" (as strings)
- Numeric values should be valid numbers
- Duration values should use Go duration format (e.g., "30s", "5m", "1h")
- Array values should be properly formatted according to the expected input
- Logging levels should be valid strings: "debug", "info", "warn", "error"

For example, `BR_TERMINATE_SESSIONS` expects "true" or "false" as a string, not a boolean value.

### Configuration Conflicts
When both environment variables and configuration files specify values for the same parameter, ensure you understand the priority hierarchy. Environment variables always take precedence over file-based configurations.

If experiencing unexpected behavior, check for conflicting values between:
- Environment variables and app.yaml settings
- Environment variables and secret.yaml settings
- Multiple environment variables affecting the same functionality

Use logging output to verify which values are actually being used by the application.

### Logging Configuration Issues
For logging-related problems:
- Verify BR_LOG_* variables are set correctly
- Check that BR_OUTPUT_FORMAT is set to "json" or "text"
- Ensure BR_LOG_OUTPUT is one of "stdout", "stderr", or "file"
- Validate file permissions if using file output
- Check log file path exists and is writable

**Section sources**
- [config.go](file://internal/config/config.go#L400-L600)

## Best Practices for CI/CD and Development

### CI/CD Pipeline Configuration
For CI/CD pipelines, follow these best practices:

1. **Use Environment-Specific Variables**: Define different environment variables for development, staging, and production environments.

2. **Secure Secret Management**: Store sensitive information like access tokens and passwords in secure secret management systems rather than hardcoding them in pipeline configurations.

3. **Consistent Naming Conventions**: Maintain consistent naming conventions across all pipeline jobs to avoid confusion.

4. **Validation Before Execution**: Implement pre-execution validation to check for required environment variables before starting the runner.

5. **Logging Configuration**: Configure appropriate logging levels based on the environment (more verbose in development, less in production).

6. **Output Format Selection**: Choose appropriate output formats for different environments (JSON for machine processing, text for human readability).

### Local Development Setup
For local development, consider these practices:

1. **Environment Variable Files**: Use .env files to manage environment variables locally, but ensure they are excluded from version control.

2. **Script Automation**: Create shell scripts to set up the development environment consistently across team members.

3. **Default Values**: Leverage default values in the configuration system to minimize the number of required environment variables for local testing.

4. **Isolated Testing**: Use isolated test environments to prevent accidental modifications to production systems.

5. **Documentation**: Maintain clear documentation of all required environment variables and their purposes.

### Example Development Script
```bash
#!/bin/bash
# setup-dev-env.sh

# Set basic configuration
export BR_ACTOR="dev-user"
export BR_ENV="development"
export BR_GITEAURL="https://gitea-dev.example.com"
export BR_REPOSITORY="organization/project-dev"
export BR_ACCESS_TOKEN="dev-token"
export BR_OUTPUT_FORMAT="text"

# Set logging configuration for development
export BR_LOG_LEVEL="debug"
export BR_LOG_FORMAT="text"
export BR_LOG_OUTPUT="stderr"

# Set SonarQube configuration for development
export SONARQUBE_URL="https://sonarqube-dev.example.com"
export SONARQUBE_TOKEN="dev-sonar-token"

# Set default command for development
export BR_COMMAND="analyze-project"

echo "Development environment configured"
```

### Production Deployment Considerations
For production deployments:
- Use BR_OUTPUT_FORMAT="json" for machine-readable logs
- Set BR_LOG_LEVEL="info" or "warn" for reduced verbosity
- Use BR_LOG_OUTPUT="stderr" to separate application logs from stdout
- Configure file logging with appropriate rotation settings
- Monitor log file permissions and disk space

Following these best practices ensures consistent and reliable operation of the apk-ci across different environments while maintaining security and ease of use.

**Section sources**
- [config.go](file://internal/config/config.go#L130-L200)
- [action.yaml](file://config/action.yaml#L1-L120)
- [providers.go](file://internal/di/providers.go#L36-L51)