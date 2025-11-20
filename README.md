# Sonatype Resource Automation (Go)

An asynchronous API service for managing Nexus Repository Manager and Sonatype IQ Server resources — repositories, privileges, roles, and user permissions — at scale.

This service exposes a small HTTP API to submit batches of repository create/delete requests. Each request is processed asynchronously and **idempotently** and returns a job ID so you can poll for status while work runs in the background.

## Table of contents

- [Features](#features)
- [Quick start](#quick-start)
- [API](#api-endpoints)
- [Architecture & design](#architecture--design)
- [Configuration Guide](#configuration-guide)
- [Development & Building](#development--building)
- [Runbook & maintenance](#runbook--maintenance)
- [Troubleshooting & Logging](#troubleshooting--logging)
- [Contributing](#contributing)
- [License](#license)

## Features

- Batch repository creation and deletion (Nexus)
- Owner role management in Sonatype IQ Server
- Idempotent operations and safe role cleanup
- Asynchronous job processing with job status polling
- Pluggable package manager configs (add formats in JSON)

## Quick start

Prerequisites:

- Go 1.21+ (or the Go version used by this repo)
- `make` (optional, for convenience)

1.  Copy the example env into `config/.env` and fill in credentials:

```bash
cp config/.env.example config/.env
# Edit values (NEXUS_URL, NEXUS_USERNAME, NEXUS_PASSWORD, IQSERVER_URL, etc.)
```

2.  Ensure `config/organizations.json` and `config/packageManager.json` contain the mappings you need. Example files are provided in `config/`.

3.  Run the server locally:

```bash
go run main.go
```

The server will start on the host/port defined in `config/.env` (defaults to `127.0.0.1:5000`).

### Build

```bash
# Build a local binary
make build

# Or build cross-platform
make all
```

## Debugging Tips

- Enable debug logs for detailed traces:

```bash
export LOG_LEVEL=DEBUG
go run main.go
```

- Key components and logs to investigate:
  - `nexus_creator`, `nexus_cleaner` — Nexus operations
  - `iq_deletion` / `iq_client` — IQ Server operations
  - `batch_manager` / `job_progress_tracker` — Job lifecycle and worker failures

## Adding a New Package Manager

1.  Edit `config/packageManager.json` and add a new object entry for the new format. The entry should include `defaultURL`, `apiEndpoint.path` and optionally `formatSpecificConfig` for Nexus specific fields.

2.  Start the server and use a test request to verify repository creation via the new API endpoint.

3.  Add unit tests in `internal/client` or `internal/service` where applicable.

## Testing & CI

- Run all tests locally to check for regressions:

```bash
go test ./...
```

- Tests should cover both **happy paths** and **error-handling** for Nexus and IQ interactions. Use mocks to simulate upstream responses.

## Troubleshooting Checklist

When something fails:

1.  Check `app.log` for stack traces and failure details.
2.  Verify credentials in `config/.env` and upstream network connectivity to Nexus/IQ.
3.  Verify organization UUIDs in `config/organizations.json`.
4.  For flaky network issues, consider increasing timeouts or adding retry logic with backoff.

## Monitoring & Metrics (Recommendations)

- Instrument the service (e.g., Prometheus) to track:
  - Job success/failure counts
  - HTTP call latency to Nexus/IQ
  - Number of active workers and queue length

## Maintenance Checklist (Quick)

Use this checklist for common maintenance tasks during deployments and on-call rotations.

1.  Verify config values and secrets in `config/.env`.
2.  Validate `config/organizations.json` and `config/packageManager.json` contain required mappings.
3.  Deploy service and confirm it starts without errors.
4.  Check the health endpoint and ensure the server responds.
5.  Tail logs and look for unexpected errors or high error rates.
6.  If the JobStore is reset, re-run any necessary job submission tests.

## Credential Rotation

1.  Update credentials in `config/.env` for Nexus, IQ, or API token.
2.  Restart the service (or use rolling restart in clustered environments).
3.  Confirm successful API calls to Nexus/IQ in logs and that jobs run successfully.

## Handling a Production Incident (high level)

1.  Check the service status and logs for obvious failures.
2.  Roll back to a previous working release if a recent change introduced a regression.
3.  If a user is impacted, create a short incident note with the job ID and failing request details.
4.  If the job store was wiped, notify users that job history was lost and suggest re-submitting their requests.

## API Endpoints

All endpoints require an Authorization header:

```
Authorization: Bearer <YOUR_API_TOKEN>
```

1. Create repositories (async):

```http
POST /repositories
```

Example request body:

```json
{
  "Requests": [
    {
      "OrganizationName": "Department A",
      "LdapUsername": "john.doe",
      "PackageManager": "npm",
      "Shared": false,
      "AppID": "my-app-001"
    }
  ]
}
```

2. Delete repositories (async):

```http
DELETE /repositories
```

The request body follows the same format as `POST /repositories`.

> **Note:** For `DELETE /repositories` the API validates the payload strictly: a delete request may either target a specific repository (`Shared=false`, `AppID` required, `PackageManager` required) or perform an offboarding-style cleanup (`Shared=true`, `AppID` required, `PackageManager` must be empty). A `DELETE` with `Shared=true` and an empty `AppID` is rejected by the API; use the offboarding flow to remove shared access, clean up app artifacts, and automatically revoke the Owner role in the associated IQ Server organization.

3. Get a job status (polling):

```http
GET /jobs/:jobID
```

The `GET` returns the job object with totals and any failed requests. Response field names are `camelCase`.

Example `curl` usage (create):

```bash
curl -X POST \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer $API_TOKEN" \
	-d '{"Requests":[{"OrganizationName":"Department A","LdapUsername":"john.doe","PackageManager":"npm","Shared":false,"AppID":"my-app-001"}]}' \
	http://127.0.0.1:5000/repositories
```

## Maintainers Guide

This section contains clear, actionable instructions for maintainers and operators. For the user-facing API reference, see `docs/user.en.md`.

## Project Overview

**Sonatype Resource Automation (Go)** is an asynchronous API service designed to manage the lifecycle of repositories, privileges, roles, and user permissions across **Nexus Repository Manager** and **Sonatype IQ Server**.

**Key Capabilities:**

- **Batch Processing**: Handles multiple creation/deletion requests in a single HTTP call.
- **Asynchronous Execution**: Returns a Job ID immediately while processing continues in the background.
- **Idempotency**: Safe to re-run operations; existing resources are skipped or updated non-destructively.
- **Smart Role Management**: Ensures users never lose access during complex deletion scenarios by analyzing remaining permissions.

## Architecture & Design

The application follows a **Clean Architecture** approach, separating concerns into distinct layers:

1.  **Transport Layer (`internal/server`)**: Handles HTTP requests, validation, and routing using `Gin`.
2.  **Orchestration Layer (`internal/server/sync.go`)**: Contains the `BatchManager` which manages the async job queue and coordinates between services.
3.  **Service Layer (`internal/service`)**: Contains the core business logic (Creation, Deletion, Role Decisions).
4.  **Client Layer (`internal/client`)**: Wraps external API calls to Nexus and IQ Server using `Resty`.
5.  **Configuration (`internal/config`)**: Centralized config loading using `Viper` and data models.

## Directory Structure

```text
.
├── bin/                     # Compiled binaries
├── config/                  # Configuration files (.env, JSON mappings)
├── docs/                    # User and maintenance documentation
├── internal/
│   ├── client/              # HTTP clients for Nexus and IQ Server
│   ├── config/              # Config loading, validation, and Struct definitions
│   ├── server/              # HTTP handlers, Router, and Batch Manager
│   ├── service/             # Business logic (Creation, Deletion, Role Engine)
│   └── utils/               # Logging (Zap) and helper functions
├── main.go                  # Entry point
└── Makefile                 # Build scripts
```

## Core Components

### 1. Entry Point (`main.go`)

- Initializes the **Zap** logger.
- Loads configuration via `internal/config`.
- Initializes the in-memory `JobStore`.
- Sets up the `BatchManager` and `Router`.
- Starts the HTTP server with graceful shutdown handling (SIGINT/SIGTERM).

### 2. Configuration (`internal/config`)

- **`Load()`**: Reads `.env`, `organizations.json`, and `packageManager.json`.
- **`CreateOpConfig`**: Converts a raw API request into an executable `OperationConfig`. This step resolves the Organization ID and determines the correct repository/role naming conventions.
- **Models**: Defines `RepositoryRequest`, `Job`, and `OperationConfig`.

Example `RepositoryRequest` payload format that the server expects (note the exact JSON property names):

```json
{
  "Requests": [
    {
      "OrganizationName": "Department A",
      "LdapUsername": "john.doe",
      "PackageManager": "npm",
      "Shared": false,
      "AppID": "my-app-001"
    }
  ]
}
```

Required fields: `OrganizationName`, `LdapUsername`, `PackageManager`. `Shared` and `AppID` are validated for compatibility: `AppID` must be set when `Shared` is false and must be omitted when `Shared` is true.

### 3. Server Layer (`internal/server`)

- **`BatchManager`**: The heart of the async engine.
  - Validates requests immediately.
  - Spawns a background goroutine for the batch.
  - Fans out processing (concurrent workers per request).
  - Aggregates results and updates the `JobStore`.
- **`Handlers`**:
  - `POST /repositories`: Validates input, enqueues job, returns 202 Accepted.
  - `GET /jobs/:id`: Polling endpoint for job status.

Example `GET /jobs/:id` response (full job payload):

```json
{
  "id": "job-123",
  "status": "completed",
  "action": "create",
  "createdAt": "2025-11-20T19:00:00Z",
  "updatedAt": "2025-11-20T19:01:23Z",
  "totalRequests": 10,
  "successfulOperations": 9,
  "failedOperations": 1,
  "notProcessedOperations": 0,
  "failedRequests": [
    {
      "request": {
        "organizationName": "Department A",
        "ldapUsername": "john.doe",
        "packageManager": "npm",
        "shared": true,
        "appId": ""
      },
      "reason": "Repository already exists"
    }
  ],
  "message": "Completed with 1 failures"
}
```

### 4. Service Layer (`internal/service`)

- **`CreationManager`**: Orchestrates the creation flow (Repo -> Privilege -> Role -> User).
- **`DeletionManager`**: Orchestrates the deletion flow based on whether the repo is shared or specific.
- **`RoleDecisionEngine`**: Contains the logic to determine what roles a user should retain when a specific role is removed.
- **`IQDeletionManager`**: Handles the removal of User Owner roles from IQ Server organizations.

## Key Logic Workflows

### 1. Repository Naming Convention

The system automatically generates names based on the configuration:

- **Format**: `{package_manager}-release-{suffix}`
- **Shared**: `npm-release-shared`
- **App Specific**: `npm-release-my-app-001`

### 2. Async Job Processing

1.  **Validation**: The API validates payload structure, organization existence, and package manager support **synchronously**.
2.  **Queueing**: A Job ID is created in memory.
3.  **Execution**:
    - Resources (Nexus) are processed.
    - If Nexus operations succeed, IQ Server operations are attempted.
    - Failures are recorded individually; one failure does not stop the batch.

### 3. Smart Role Cleanup (`RoleDecisionEngine`)

When a repository is deleted, the associated role is removed from the user. The system calculates the new role set to prevent locking the user out.

**Logic Flow:**

1.  **Identify Remaining Roles**: Calculate what the user would have after the target role is removed.
2.  **Check for "Active" Roles**: Does the user still have specific application roles? (Ignoring `BASE_ROLE`, `EXTRA_ROLE`, and `repositories.share`).
    - **YES**: The user is still active on other apps. Keep `EXTRA_ROLES`.
    - **NO**: The user has no specific apps left. Remove `EXTRA_ROLES`.
3.  **Safety Fallback**: If the resulting role list is empty, assign the `BASE_ROLE` (defined in `.env`) to ensure the user can still login.

### 4. IQ Server Integration

- **Creation**: Adds the "Owner" role to the user for the specific Organization ID defined in `organizations.json`.
- **Deletion**: Checks if the user has any other roles relevant to that organization before revoking the "Owner" role.

## Configuration Guide

### Environment Variables (`config/.env`)

| Variable     | Description                                 | Example                          |
| :----------- | :------------------------------------------ | :------------------------------- |
| `NEXUS_URL`  | Nexus API Base URL                          | `http://nexus:8081/service/rest` |
| `EXTRA_ROLE` | Roles added to every user (comma-separated) | `role1,role2`                    |
| `BASE_ROLE`  | Fallback role if user has no other access   | `nx-admin`                       |
| `LOG_LEVEL`  | Logging verbosity                           | `DEBUG`, `INFO`, `WARN`          |
| `API_HOST`   | Host address to bind the server             | `127.0.0.1`                      |
| `PORT`       | Port to run the server on                   | `5000`                           |

### Default Configuration

The application uses the following default timeouts (defined in code):

- **Read Timeout**: 15 seconds
- **Write Timeout**: 15 seconds
- **Idle Timeout**: 60 seconds
- **Shutdown Timeout**: 5 seconds

### Package Manager Config (`config/packageManager.json`)

This file drives the logic for creating repositories. You can add support for new formats without changing code.

```json
"npm": {
  "defaultURL": "https://registry.npmjs.org",
  "apiEndpoint": {
    "path": "/v1/repositories/npm/proxy",
    "formatSpecificConfig": {
      "npm": { "removeQuarantined": false }
    }
  }
}
```

### Organizations (`config/organizations.json`)

Maps human-readable names to IQ Server UUIDs.

````json
{
  "Department A": "7b2f3034e08445fe9bb02ce5565f98b5"
}```


## Development & Building

### Prerequisites

-   Go 1.21+
-   Make (optional, for build scripts)

### Adding a New Package Manager

1.  Open `config/packageManager.json`.
2.  Add a new key (e.g., `pypi`).
3.  Define the `defaultURL` (upstream proxy).
4.  Define the `apiEndpoint` path (Nexus API endpoint for that format).
5.  Add any format-specific JSON fields required by the Nexus API in `formatSpecificConfig`.

### Compile Binaries

Use the included `Makefile` to build for multiple platforms:

```bash
make all
# Outputs:
# bin/sra-darwin-arm64
# bin/sra-linux-amd64
# bin/sra-windows-amd64.exe
````

### Running Locally

```bash
go run main.go
```

The server defaults to `127.0.0.1:5000` (configurable in `.env`).

## Runbook & maintenance

This short runbook covers common troubleshooting and maintenance tasks.

### Start (local/dev)

```bash
cp config/.env.example config/.env
# Edit config/.env as needed
go run main.go
```

### Health & verification

Check the health endpoint and logs:

```bash
curl -s -H "Authorization: Bearer $API_TOKEN" http://127.0.0.1:5000/health
tail -n 200 app.log
```

### Verify created Nexus artifacts

Use the Nexus API endpoints to confirm created resources after a successful job. Examples:

```bash
# Check repository
curl -u $NEXUS_USER:$NEXUS_PASS $NEXUS_URL/v1/repositories/<repoName>

# Check role
curl -u $NEXUS_USER:$NEXUS_PASS $NEXUS_URL/v1/security/roles/<roleName>

# Check user and roles
curl -u $NEXUS_USER:$NEXUS_PASS "$NEXUS_URL/v1/security/users?userId=<username>"
```

### Verify IQ Server role assignment

```bash
curl -u $IQ_USER:$IQ_PASS "$IQSERVER_URL/api/v2/roleMemberships/organization/<organizationId>/user/<username>"
```

### Restart and job store

System-managed service (example):

```bash
systemctl restart sra.service
```

Local restart just stops and starts the process. Note: the `JobStore` is in-memory; restarting clears job history. To preserve job history use a persistent store (e.g., Redis or a database) and update `JobStore` implementation.

## Troubleshooting & Logging

### Logs

The application uses **Zap** for structured logging.

- **Console**: Human-readable output.
- **File (`app.log`)**: JSON formatted output for ingestion/parsing.

**Log Levels:**

- `INFO`: High-level operation success/failure (default).
- `DEBUG`: Detailed request tracing, HTTP payloads, and logic flows (Enable via `LOG_LEVEL=DEBUG`).

### Common Issues

**1. "Job not found" (404)**

- **Cause**: The application was restarted.
- **Reason**: `JobStore` is currently in-memory only. Restarting the service clears all job history.

**2. Nexus/IQ Authentication Errors**

- **Check**: `.env` credentials.
- **Logs**: Look for `HTTP 401` or `HTTP 403` in `app.log`.

**3. "Organization not found"**

- **Check**: Ensure the exact string sent in the JSON request matches a key in `config/organizations.json`.

**4. User Roles Not Updating Correctly**

- **Debug**: Enable `LOG_LEVEL=DEBUG`. Look for logs from component `nexus_creator` or `nexus_cleaner`. The logs will detail exactly which roles were detected, deduplicated, and finally applied.
