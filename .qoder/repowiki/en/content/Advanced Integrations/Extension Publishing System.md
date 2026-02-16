# Extension Publishing System

<cite>
**Referenced Files in This Document**
- [README.md](file://README.md)
- [main.go](file://cmd/benadis-runner/main.go)
- [extension_publish.go](file://internal/app/extension_publish.go)
- [extension-publish.md](file://docs/epics/extension-publish.md)
- [external-extension-workflow.md](file://docs/diagrams/external-extension-workflow.md)
- [gitea.go](file://internal/entity/gitea/gitea.go)
- [interfaces.go](file://internal/entity/gitea/interfaces.go)
- [constants.go](file://internal/constants/constants.go)
- [config.go](file://internal/config/config.go)
- [app.yaml](file://config/app.yaml)
</cite>

## Table of Contents
1. [Introduction](#introduction)
2. [System Architecture](#system-architecture)
3. [Core Components](#core-components)
4. [Extension Publishing Workflow](#extension-publishing-workflow)
5. [Gitea API Integration](#gitea-api-integration)
6. [Configuration Management](#configuration-management)
7. [Error Handling and Reporting](#error-handling-and-reporting)
8. [Testing Strategy](#testing-strategy)
9. [Deployment and Usage](#deployment-and-usage)
10. [Troubleshooting Guide](#troubleshooting-guide)

## Introduction

The Extension Publishing System is a sophisticated automation framework designed to streamline the distribution of 1C:Enterprise external extensions across multiple subscribed repositories. This system automates the entire process of extension updates, from detecting new releases to creating pull requests in target repositories, ensuring consistent and reliable deployment across organizational infrastructure.

The system operates on a subscription-based model where target repositories maintain special "subscription branches" that indicate their interest in receiving extension updates. When a new release is published in the source extension repository, the system automatically discovers all subscribing repositories and propagates the updated extension files through automated pull requests.

## System Architecture

The Extension Publishing System follows a modular architecture with clear separation of concerns:

```mermaid
graph TB
subgraph "User Interface Layer"
CLI[CLI Commands]
Actions[Gitea Actions]
end
subgraph "Application Layer"
Main[Main Application]
ExtensionPublish[Extension Publisher]
ConfigLoader[Configuration Loader]
end
subgraph "Business Logic Layer"
SubscriberFinder[Subscriber Finder]
SyncEngine[Sync Engine]
PREngine[PR Engine]
Reporter[Reporting Engine]
end
subgraph "Integration Layer"
GiteaAPI[Gitea API Client]
GitOps[Git Operations]
end
subgraph "Data Layer"
ConfigStore[Configuration Store]
LogStore[Log Storage]
end
CLI --> Main
Actions --> Main
Main --> ExtensionPublish
ExtensionPublish --> ConfigLoader
ExtensionPublish --> SubscriberFinder
ExtensionPublish --> SyncEngine
ExtensionPublish --> PREngine
ExtensionPublish --> Reporter
SubscriberFinder --> GiteaAPI
SyncEngine --> GiteaAPI
PREngine --> GiteaAPI
GiteaAPI --> GitOps
ConfigLoader --> ConfigStore
Reporter --> LogStore
```

**Diagram sources**
- [main.go](file://cmd/benadis-runner/main.go#L16-L262)
- [extension_publish.go](file://internal/app/extension_publish.go#L979-L1253)

The architecture consists of several key layers:

- **Presentation Layer**: CLI interface and Gitea Actions integration
- **Application Layer**: Central command routing and configuration management
- **Business Logic Layer**: Core publishing algorithms and orchestration
- **Integration Layer**: Gitea API connectivity and Git operations
- **Data Layer**: Configuration persistence and logging

## Core Components

### Main Application Entry Point

The application entry point serves as the central command router, handling various operational modes and delegating to appropriate subsystems based on the selected command.

```mermaid
flowchart TD
Start([Application Start]) --> LoadConfig[Load Configuration]
LoadConfig --> ParseCommand[Parse Command]
ParseCommand --> CheckCommand{Command Type?}
CheckCommand --> |extension-publish| CallPublisher[Call ExtensionPublish]
CheckCommand --> |other commands| CallOther[Call Other Handler]
CallPublisher --> ValidateConfig[Validate Configuration]
ValidateConfig --> FindSubscribers[Find Subscribers]
FindSubscribers --> ProcessSubscribers[Process Each Subscriber]
ProcessSubscribers --> CreatePR[Create Pull Request]
CreatePR --> GenerateReport[Generate Report]
GenerateReport --> End([Complete])
CallOther --> End
End --> Exit([Exit Application])
```

**Diagram sources**
- [main.go](file://cmd/benadis-runner/main.go#L30-L260)

**Section sources**
- [main.go](file://cmd/benadis-runner/main.go#L16-L262)

### Extension Publishing Engine

The core publishing engine orchestrates the complete extension distribution process, implementing sophisticated algorithms for subscriber discovery, file synchronization, and pull request creation.

```mermaid
sequenceDiagram
participant Source as Source Repository
participant System as Extension System
participant Target as Target Repository
participant Gitea as Gitea API
Source->>System : New Release Published
System->>Gitea : Get Latest Release Info
Gitea-->>System : Release Details
System->>Gitea : Find All Organizations
Gitea-->>System : Organization List
loop For Each Organization
System->>Gitea : Search Repositories
Gitea-->>System : Repository List
loop For Each Repository
System->>Gitea : Check Subscription Branch
Gitea-->>System : Branch Exists?
alt Branch Exists
System->>Gitea : Get Source Files
Gitea-->>System : File Contents
System->>Gitea : Get Target Files
Gitea-->>System : Target Contents
System->>Gitea : Create Branch + Commit
Gitea-->>System : Commit Created
System->>Gitea : Create Pull Request
Gitea-->>System : PR Created
end
end
end
System->>System : Generate Final Report
```

**Diagram sources**
- [extension_publish.go](file://internal/app/extension_publish.go#L979-L1253)

**Section sources**
- [extension_publish.go](file://internal/app/extension_publish.go#L456-L567)

### Subscription Management System

The subscription management system maintains a registry of interested parties through a unique branch naming convention that enables automatic discovery of target repositories.

```mermaid
classDiagram
class SubscribedRepo {
+string Organization
+string Repository
+string TargetBranch
+string TargetDirectory
+string SubscriptionBranch
}
class SubscriptionBranchParser {
+ParseSubscriptionBranch(branchName) SubscribedRepo
+IsSubscriptionBranch(branchName) bool
}
class SubscriberFinder {
+FindSubscribedRepos(l, api, sourceRepo, extensions) []SubscribedRepo
+generateBranchPatterns(extensions) []string
}
class BranchPatternGenerator {
+generatePattern(org, repo, extDir) string
}
SubscriptionBranchParser --> SubscribedRepo : creates
SubscriberFinder --> SubscriptionBranchParser : uses
SubscriberFinder --> BranchPatternGenerator : uses
SubscriberFinder --> SubscribedRepo : returns
```

**Diagram sources**
- [extension_publish.go](file://internal/app/extension_publish.go#L60-L148)

**Section sources**
- [extension_publish.go](file://internal/app/extension_publish.go#L150-L267)

## Extension Publishing Workflow

### Subscription Branch Format

The system uses a standardized branch naming convention to identify subscription targets:

```
{Organization}_{Repository}_{ExtensionDirectory}
```

Where:
- **Organization**: Target repository's organization name
- **Repository**: Target repository name  
- **ExtensionDirectory**: Path to extension directory within target project

**Example**: `APKHolding_ERP_cfe_CommonExt` indicates the `APKHolding/ERP` repository wants updates for the `cfe/CommonExt` extension directory.

### File Synchronization Process

The synchronization process ensures complete parity between source and target repositories:

```mermaid
flowchart TD
Start([Start Sync Process]) --> GetSourceFiles[Get Source Files]
GetSourceFiles --> GetTargetFiles[Get Target Files Map]
GetTargetFiles --> GenerateBranch[Generate Branch Name]
GenerateBranch --> CompareFiles[Compare Files]
CompareFiles --> SourceExists{File Exists in Source?}
SourceExists --> |Yes| CheckTargetExists{File Exists in Target?}
SourceExists --> |No| DeleteFile[Delete from Target]
CheckTargetExists --> |Yes| CheckContent{Content Changed?}
CheckTargetExists --> |No| CreateFile[Create in Target]
CheckContent --> |Yes| UpdateFile[Update in Target]
CheckContent --> |No| SkipFile[Skip File]
DeleteFile --> NextFile[Next File]
CreateFile --> NextFile
UpdateFile --> NextFile
SkipFile --> NextFile
NextFile --> MoreFiles{More Files?}
MoreFiles --> |Yes| CompareFiles
MoreFiles --> |No| CreateCommit[Create Commit]
CreateCommit --> End([Sync Complete])
```

**Diagram sources**
- [extension_publish.go](file://internal/app/extension_publish.go#L456-L567)

**Section sources**
- [extension_publish.go](file://internal/app/extension_publish.go#L269-L418)

### Pull Request Generation

Each successful synchronization triggers automated pull request creation with comprehensive metadata:

```mermaid
classDiagram
class PRGenerator {
+BuildExtensionPRTitle(extName, version) string
+BuildExtensionPRBody(release, sourceRepo, extName, releaseURL) string
+CreateExtensionPR(l, api, syncResult, release, extName, sourceRepo, releaseURL) PRResponse
}
class PRMetadata {
+string Title
+string Body
+string Head
+string Base
+int64 Number
+string HTMLURL
}
class ReleaseInfo {
+string TagName
+string Name
+string Body
+string HTMLURL
+time CreatedAt
}
PRGenerator --> PRMetadata : creates
PRGenerator --> ReleaseInfo : uses
PRGenerator --> PRMetadata : returns
```

**Diagram sources**
- [extension_publish.go](file://internal/app/extension_publish.go#L625-L688)

**Section sources**
- [extension_publish.go](file://internal/app/extension_publish.go#L569-L624)

## Gitea API Integration

### API Client Architecture

The system integrates deeply with Gitea's REST API through a comprehensive client implementation that supports all required operations for extension publishing.

```mermaid
classDiagram
class GiteaAPI {
+string GiteaURL
+string Owner
+string Repo
+string AccessToken
+string BaseBranch
+string NewBranch
+string Command
+NewGiteaAPI(config) API
+sendReq(urlString, reqBody, respType) Response
}
class APIInterface {
<<interface>>
+GetLatestRelease() Release
+GetReleaseByTag(tag) Release
+SearchOrgRepos(orgName) []Repository
+HasBranch(owner, repo, branch) bool
+GetRepositoryContents(filepath, branch) []FileInfo
+AnalyzeProject(branch) []string
+SetRepositoryStateWithNewBranch(l, operations, branch, newBranch, message) string
+CreatePRWithOptions(opts) PRResponse
}
class Release {
+int64 ID
+string TagName
+string Name
+string Body
+[]ReleaseAsset Assets
+string CreatedAt
+string PublishedAt
}
class Repository {
+int64 ID
+string Name
+string FullName
+RepositoryOwner Owner
+string DefaultBranch
+bool Private
+bool Fork
}
GiteaAPI ..|> APIInterface : implements
GiteaAPI --> Release : manages
GiteaAPI --> Repository : manages
```

**Diagram sources**
- [gitea.go](file://internal/entity/gitea/gitea.go#L288-L317)
- [interfaces.go](file://internal/entity/gitea/interfaces.go#L18-L57)

**Section sources**
- [gitea.go](file://internal/entity/gitea/gitea.go#L1182-L1200)
- [interfaces.go](file://internal/entity/gitea/interfaces.go#L18-L57)

### Authentication and Security

The system implements robust authentication mechanisms for secure Gitea API access:

- **Token-Based Authentication**: Uses bearer tokens for API requests
- **Environment Variable Management**: Secure token storage through environment variables
- **Rate Limiting**: Built-in handling for API rate limits and retry logic
- **Error Handling**: Comprehensive error propagation with meaningful error messages

**Section sources**
- [gitea.go](file://internal/entity/gitea/gitea.go#L469-L503)

## Configuration Management

### Multi-Layer Configuration System

The system employs a hierarchical configuration approach supporting environment variables, YAML files, and default values:

```mermaid
flowchart TD
Environment[Environment Variables] --> Priority1[Priority 1]
YAMLConfig[YAML Configuration Files] --> Priority2[Priority 2]
Defaults[Default Values] --> Priority3[Priority 3]
Priority1 --> Merge[Configuration Merge]
Priority2 --> Merge
Priority3 --> Merge
Merge --> Apply[Apply to System]
Apply --> Runtime[Runtime Configuration]
```

**Diagram sources**
- [config.go](file://internal/config/config.go#L548-L702)

### Key Configuration Parameters

| Parameter | Purpose | Environment Variable | Default Value |
|-----------|---------|---------------------|---------------|
| `BR_COMMAND` | Command selection | `BR_COMMAND` | Empty |
| `GITEA_TOKEN` | API authentication | `GITHUB_TOKEN` | None |
| `GITHUB_REPOSITORY` | Source repository | `GITHUB_REPOSITORY` | None |
| `GITHUB_REF_NAME` | Release tag | `GITHUB_REF_NAME` | `main` |
| `BR_EXT_DIR` | Extension directory | `BR_EXT_DIR` | Auto-detected |
| `BR_DRY_RUN` | Test mode | `BR_DRY_RUN` | `false` |

**Section sources**
- [config.go](file://internal/config/config.go#L128-L209)
- [app.yaml](file://config/app.yaml#L1-L138)

## Error Handling and Reporting

### Comprehensive Error Management

The system implements structured error handling with detailed reporting capabilities:

```mermaid
classDiagram
class PublishResult {
+SubscribedRepo Subscriber
+PublishStatus Status
+SyncResult SyncResult
+int PRNumber
+string PRURL
+error Error
+string ErrorMessage
+int64 DurationMs
}
class PublishReport {
+string ExtensionName
+string Version
+string SourceRepo
+time StartTime
+time EndTime
+[]PublishResult Results
+SuccessCount() int
+FailedCount() int
+SkippedCount() int
+HasErrors() bool
+TotalDuration() time.Duration
}
class PublishStatus {
<<enumeration>>
+success
+failed
+skipped
}
PublishReport --> PublishResult : contains
PublishResult --> PublishStatus : uses
```

**Diagram sources**
- [extension_publish.go](file://internal/app/extension_publish.go#L745-L849)

### Reporting Formats

The system supports dual reporting formats:

**JSON Output** (`BR_OUTPUT_JSON=true`):
- Machine-readable structured data
- Complete execution statistics
- Error details for automation

**Human-Readable Output**:
- Formatted console logs
- Summary statistics
- Individual subscriber status

**Section sources**
- [extension_publish.go](file://internal/app/extension_publish.go#L869-L977)

## Testing Strategy

### Comprehensive Test Coverage

The system implements extensive testing across multiple layers:

```mermaid
graph TB
subgraph "Unit Tests"
UT_API[Gitea API Unit Tests]
UT_Publish[Extension Publish Unit Tests]
UT_Utils[Utility Function Tests]
end
subgraph "Integration Tests"
IT_API[Gitea API Integration Tests]
IT_Publish[End-to-End Publish Tests]
IT_Config[Configuration Tests]
end
subgraph "Mock Infrastructure"
MockGitea[Mock Gitea Server]
MockFS[Mock File System]
TestDB[Test Database]
end
UT_API --> MockGitea
UT_Publish --> MockGitea
IT_Publish --> MockGitea
IT_API --> MockGitea
MockGitea --> TestDB
MockFS --> TestDB
```

**Diagram sources**
- [extension_publish_test.go](file://internal/app/extension_publish_test.go)
- [extension_publish_integration_test.go](file://internal/app/extension_publish_integration_test.go)

### Test Categories

- **Unit Tests**: Individual function and method testing
- **Integration Tests**: Cross-component functionality validation
- **Mock Testing**: External dependency simulation
- **Regression Tests**: Prevent functionality degradation

**Section sources**
- [extension_publish_test.go](file://internal/app/extension_publish_test.go)
- [extension_publish_integration_test.go](file://internal/app/extension_publish_integration_test.go)

## Deployment and Usage

### Gitea Actions Integration

The system seamlessly integrates with Gitea Actions for automated extension publishing:

```yaml
name: Publish Extension
on:
  release:
    types: [published]

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - name: Publish extension to subscribers
        uses: docker://your-registry/benadis-runner:latest
        env:
          BR_COMMAND: extension-publish
          GITEA_TOKEN: ${{ secrets.GITEA_TOKEN }}
          GITHUB_REPOSITORY: ${{ github.repository }}
```

### Manual Execution

For manual execution scenarios:

```bash
# Configure environment
export BR_COMMAND=extension-publish
export GITEA_TOKEN=your_access_token
export GITHUB_REPOSITORY=organization/repository

# Execute publisher
./benadis-runner extension-publish
```

**Section sources**
- [extension-publish.md](file://docs/epics/extension-publish.md#L299-L326)

## Troubleshooting Guide

### Common Issues and Solutions

| Issue | Symptoms | Solution |
|-------|----------|----------|
| **Authentication Failure** | API returns 401/403 errors | Verify GITEA_TOKEN validity and permissions |
| **Subscription Not Found** | No subscribers detected | Check branch naming format `{Org}_{Repo}_{Dir}` |
| **File Synchronization Errors** | Partial updates or missing files | Verify source repository structure and permissions |
| **Pull Request Creation Failures** | PR not created despite successful sync | Check target repository branch protection rules |

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
export LOG_LEVEL=debug
export BR_OUTPUT_JSON=true
./benadis-runner extension-publish
```

### Monitoring and Metrics

The system provides comprehensive logging for operational visibility:

- **Execution Time**: Per-subscriber processing duration
- **Success Rates**: Overall publication success metrics  
- **Error Patterns**: Common failure categories and frequencies
- **API Usage**: Rate limiting and quota consumption

**Section sources**
- [extension_publish.go](file://internal/app/extension_publish.go#L869-L977)