# GEMINI.md

## Project Overview

This project, `apk-ci`, is a Go-based command-line tool for automating tasks related to the 1C:Enterprise platform. It provides a set of modules for performing various operations, including:

*   **Convert:** Converts data between different formats.
*   **DBRestore:** Restores and manages MSSQL databases.
*   **ServiceMode:** Manages the service mode of 1C information bases.
*   **EDT:** Integrates with 1C:Enterprise Development Tools.
*   **SonarQube:** Integrates with SonarQube for code quality scanning.

The application is designed to be modular, with each module having its own set of commands and configuration options. It uses a centralized configuration system that supports environment variables, configuration files (JSON/YAML), and default values.

## Building and Running

The project uses a `Makefile` to automate common development tasks.

### Building the application

To build the application, run the following command:

```bash
make build
```

This will create an executable file named `apk-ci` in the `build` directory.

### Running the application

To run the application, you need to specify a command and any required options. The general syntax is:

```bash
./build/apk-ci <command> [options]
```

For example, to enable the service mode for an information base named `MyInfobase`, you would run:

```bash
./build/apk-ci service-mode-enable --infobase MyInfobase
```

### Running tests

To run the tests, use the following command:

```bash
make test
```

## Development Conventions

*   **Configuration:** The application uses the `cleanenv` library for configuration management. Configuration can be provided through environment variables, configuration files, or default values.
*   **Logging:** The application uses the `slog` library for structured logging.
*   **Testing:** The application uses the `testify` library for testing.
*   **Linting:** The project uses `golangci-lint` for linting. To run the linter, use the command `make lint`.
*   **Dependencies:** Dependencies are managed using Go modules. To install or update dependencies, use the `make deps` or `make deps-update` commands.
