# API User Guide

## How it works

1. **Send a request**: You tell the API what to create or delete.
2. **Get a Job ID**: The API says "Okay, I'm working on it" and gives you an ID.
3. **Check Status**: You use that ID to check if the work is finished.

## Authentication

You need a token to use this API. Put it in the header of your request:
`Authorization: Bearer <YOUR_API_TOKEN>`

## Endpoints

### 1. Create Repositories

**URL**: `POST /repositories`

**Body**:

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

Note: The JSON key names are shown with the canonical struct field names (capitalized as used in the Go server). The decoder accepts common casing variations, but using the exact names avoids confusion.

### 2. Delete Repositories

**URL**: `DELETE /repositories`

**Body**: Same as above.

### 3. Check Job Status

**URL**: `GET /jobs/<JOB_ID>`

**Response (When Done):**

The `GET /jobs/<JOB_ID>` returns the full job object with all metrics. Example:

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

Note: Response property names return in camelCase (e.g., `jobId`, `createdAt`, `successfulOperations`).

Additional details:

- `status`: One of `pending`, `processing`, `completed`, `failed`.
- `action`: Either `create` or `delete` — indicates what the job is executing.

## Important Rules

- **OrganizationName**: Must match exactly what is in the system (case-sensitive mapping against the `config/organizations.json` key).
- **PackageManager**: e.g., `npm`, `maven`, `docker` — must exist in `config/packageManager.json`.
- **Shared**:
  - If `true` (Shared Repo): Leave `AppID` empty.
  - If `false` (App Repo): You MUST provide an `AppID`.
  - Note: The `Requests` entry requires `OrganizationName`, `LdapUsername`, and `PackageManager` to be present. The server validates `Shared` and `AppID` combinations and returns `422` if the rule is violated.

## Common Errors

- **401 Unauthorized**: You forgot the token or it's wrong.
- **422 Unprocessable Entity**: Your JSON is wrong (e.g., missing fields).
- **404 Not Found**: The Job ID doesn't exist (maybe the server restarted).
