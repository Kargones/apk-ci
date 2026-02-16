# Environment Variables

<cite>
**Referenced Files in This Document**   
- [config.go](file://internal/config/config.go)
- [sonarqube.go](file://internal/config/sonarqube.go)
- [action.yaml](file://config/action.yaml)
- [constants.go](file://internal/constants/constants.go)
</cite>

## Table of Contents
1. [Configuration Mechanism Overview](#configuration-mechanism-overview)
2. [Environment Variable Binding with Struct Tags](#environment-variable-binding-with-struct-tags)
3. [Supported Environment Variables](#supported-environment-variables)
4. [Configuration Loading Priority](#configuration-loading-priority)
5. [Practical Examples for Different Workflows](#practical-examples-for-different-workflows)
6. [Common Issues and Troubleshooting](#common-issues-and-troubleshooting)
7. [Best Practices for CI/CD and Development](#best-practices-for-cicd-and-development)

## Configuration Mechanism Overview

The benadis-runner application implements a comprehensive configuration system that prioritizes environment variables over YAML configuration files using the cleanenv library. This mechanism allows for flexible configuration management across different environments while maintaining backward compatibility with existing YAML-based configurations.

The configuration system follows a hierarchical approach where environment variables take precedence over file-based configurations, enabling dynamic runtime adjustments without modifying configuration files. The cleanenv library is used throughout the codebase to parse environment variables and bind them to struct fields, providing type safety and default value support.

The primary configuration structure is defined in the Config struct within internal/config/config.go, which serves as the central repository for all application settings. This struct integrates various configuration sources including environment variables, YAML files (app.yaml, project.yaml, secret.yaml, dbconfig.yaml), and command-line inputs from GitHub Actions.

**Section sources**
- [config.go](file://internal/config/config.go#L130-L200)

## Environment Variable Binding with Struct Tags

The benadis-runner application uses struct tags to establish the binding between environment variables and configuration fields. Each field in the Config struct is annotated with `env` and `env-default` tags that specify the corresponding environment variable name and default value.

The cleanenv library reads these struct tags during configuration loading and automatically populates the struct fields with values from environment variables when available. If an environment variable is not set, the library uses the specified default value or leaves the field empty if no default is provided.

For example, the Config struct contains fields like:
- `Actor string `env:"BR_ACTOR" env-default:"""`` - binds to BR_ACTOR environment variable
- `Env string `env:"BR_ENV" env-default:"dev"`` - binds to BR_ENV with "dev" as default
- `Command string `env:"BR_COMMAND" env-default:""`` - binds to BR_COMMAND environment variable

This approach provides a clean separation between configuration definition and implementation, allowing developers to easily understand which environment variables affect specific configuration parameters.

**Section sources**
- [config.go](file://internal/config/config.go#L130-L150)

## Supported Environment Variables

The benadis-runner supports a comprehensive set of environment variables that control various aspects of application behavior. These variables are categorized by their functional areas and have specific purposes, default values, and impacts on application execution.

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

### Configuration File Path Variables
These variables specify the locations of configuration files:

- **BR_CONFIG_SYSTEM**: Path to the system configuration file (app.yaml). No default value.
- **BR_CONFIG_PROJECT**: Path to the project configuration file (project.yaml). No default value.
- **BR_CONFIG_SECRET**: Path to the secret configuration file (secret.yaml). No default value.
- **BR_CONFIG_DBDATA**: Path to the database configuration file (dbconfig.yaml). No default value.
- **BR_CONFIG_MENU_MAIN**: Path to the main menu configuration file (menu_main.yaml). No default value.
- **BR_CONFIG_MENU_DEBUG**: Path to the debug menu configuration file (menu_debug.yaml). No default value.

### SonarQube Integration Variables
These variables configure the SonarQube integration:

- **SONARQUBE_URL**: Base URL of the SonarQube server. Default: "http://localhost:9000".
- **SONARQUBE_TOKEN**: Authentication token for SonarQube API access. Required.
- **SONARQUBE_TIMEOUT**: Timeout for SonarQube API requests. Default: 30 seconds.
- **SONARQUBE_RETRY_ATTEMPTS**: Number of retry attempts for failed API requests. Default: 3.
- **SONARQUBE_RETRY_DELAY**: Initial delay between retry attempts. Default: 5 seconds.
- **SONARQUBE_PROJECT_PREFIX**: Prefix for SonarQube project keys. Default: "benadis".
- **SONARQUBE_DEFAULT_VISIBILITY**: Default visibility for new projects ("private" or "public"). Default: "private".
- **SONARQUBE_QUALITY_GATE_TIMEOUT**: Timeout for quality gate status checks. Default: 300 seconds.
- **SONARQUBE_DISABLE_BRANCH_ANALYSIS**: Disables branch analysis for Community Edition compatibility. Default: true.

### Scanner Configuration Variables
These variables control the sonar-scanner tool:

- **SONARQUBE_SCANNER_URL**: URL to download the sonar-scanner. Default: points to version 4.8.0.2856.
- **SONARQUBE_SCANNER_VERSION**: Version of sonar-scanner to use. Default: "4.8.0.2856".
- **SONARQUBE_JAVA_OPTS**: JVM options for the scanner. Default: "-Xmx2g".
- **SONARQUBE_SCANNER_TIMEOUT**: Timeout for scanner execution. Default: 600 seconds.
- **SONARQUBE_SCANNER_WORK_DIR**: Working directory for scanner execution. Default: "/tmp/benadis".
- **SONARQUBE_SCANNER_TEMP_DIR**: Temporary directory for scanner files. Default: "/tmp/benadis/scanner/temp".

### Git Configuration Variables
These variables configure Git operations:

- **GIT_USER_NAME**: Username for Git operations. Default: "benadis-runner".
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

**Section sources**
- [config.go](file://internal/config/config.go#L130-L200)
- [sonarqube.go](file://internal/config/sonarqube.go#L15-L50)

## Configuration Loading Priority

The benadis-runner follows a specific priority order when loading configuration values, ensuring that more specific or runtime-defined values override general or static ones. The priority hierarchy from highest to lowest is:

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
To perform a database restore operation, set the following environment variables:

```bash
export BR_COMMAND=dbrestore
export BR_INFOBASE_NAME="test_database"
export BR_ACTOR="gitops-bot"
export BR_GITEAURL="https://gitea.example.com"
export BR_REPOSITORY="organization/project"
export BR_ACCESS_TOKEN="your-access-token"
export DBRESTORE_SERVER="localhost"
export DBRESTORE_DATABASE="test_database"
export DBRESTORE_BACKUP="/backups/test_database.bak"
export DBRESTORE_TIMEOUT="300s"
```

This configuration will initiate a database restore operation using the specified backup file, with the operation logged under the gitops-bot actor.

### Service Mode Management
To enable service mode for maintenance operations:

```bash
export BR_COMMAND=service-mode-enable
export BR_INFOBASE_NAME="production_db"
export BR_ACTOR="admin-user"
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
export BR_GITEAURL="https://gitea.example.com"
export BR_REPOSITORY="organization/project"
export BR_ACCESS_TOKEN="your-access-token"
```

### SonarQube Scanning
To scan a specific branch with SonarQube:

```bash
export BR_COMMAND=sq-scan-branch
export BR_ACTOR="ci-bot"
export BR_GITEAURL="https://gitea.example.com"
export BR_REPOSITORY="organization/project"
export BR_ACCESS_TOKEN="your-access-token"
export SONARQUBE_URL="https://sonarqube.example.com"
export SONARQUBE_TOKEN="your-sonarqube-token"
export INPUT_BRANCHFORSCAN="feature/new-module"
export INPUT_COMMITHASH="a1b2c3d4e5f6"
```

This configuration will trigger a SonarQube scan on the specified branch and commit, using the provided SonarQube server credentials.

**Section sources**
- [config.go](file://internal/config/config.go#L130-L200)
- [action.yaml](file://config/action.yaml#L1-L120)
- [constants.go](file://internal/constants/constants.go#L1-L200)

## Common Issues and Troubleshooting

### Incorrect Naming Conventions
One common issue is incorrect environment variable naming. Ensure that:
- All BR_* variables use uppercase letters and underscores
- SonarQube variables follow the SONARQUBE_* pattern
- Git variables use the GIT_* pattern
- RAC variables use the RAC_* pattern

For example, use `BR_COMMAND` instead of `br_command` or `BrCommand`.

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

For example, `BR_TERMINATE_SESSIONS` expects "true" or "false" as a string, not a boolean value.

### Configuration Conflicts
When both environment variables and configuration files specify values for the same parameter, ensure you understand the priority hierarchy. Environment variables always take precedence over file-based configurations.

If experiencing unexpected behavior, check for conflicting values between:
- Environment variables and app.yaml settings
- Environment variables and secret.yaml settings
- Multiple environment variables affecting the same functionality

Use logging output to verify which values are actually being used by the application.

**Section sources**
- [config.go](file://internal/config/config.go#L400-L600)

## Best Practices for CI/CD and Local Development

### CI/CD Pipeline Configuration
For CI/CD pipelines, follow these best practices:

1. **Use Environment-Specific Variables**: Define different environment variables for development, staging, and production environments.

2. **Secure Secret Management**: Store sensitive information like access tokens and passwords in secure secret management systems rather than hardcoding them in pipeline configurations.

3. **Consistent Naming Conventions**: Maintain consistent naming conventions across all pipeline jobs to avoid confusion.

4. **Validation Before Execution**: Implement pre-execution validation to check for required environment variables before starting the runner.

5. **Logging and Monitoring**: Enable appropriate logging levels based on the environment (more verbose in development, less in production).

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

# Set SonarQube configuration for development
export SONARQUBE_URL="https://sonarqube-dev.example.com"
export SONARQUBE_TOKEN="dev-sonar-token"

# Set default command for development
export BR_COMMAND="analyze-project"

echo "Development environment configured"
```

Following these best practices ensures consistent and reliable operation of the benadis-runner across different environments while maintaining security and ease of use.

**Section sources**
- [config.go](file://internal/config/config.go#L130-L200)
- [action.yaml](file://config/action.yaml#L1-L120)