# 🚀 Repository Management API User Guide

This guide details how to interact with the Repository Management API to create, delete, and manage application repositories.

---

## 🔁 The Asynchronous Flow (How it works)

This API uses an asynchronous job model for all major operations.

1.  **Send a request:** You tell the API what resource to create or delete (e.g., a repository).
2.  **Get a Job ID:** The API immediately responds with a unique `JOB_ID` and a `status: pending`.
3.  **Check Status:** You use the `JOB_ID` to poll the status until the work is finished (status is `completed` or `failed`).

---

## 🔒 Authentication

All requests **must** be authenticated. Your API token must be included in the `Authorization` header.

| Header Key      | Value Format              |
| :-------------- | :------------------------ |
| `Authorization` | `Bearer <YOUR_API_TOKEN>` |

---

## 🏗️ Endpoints

### 1. Create Repositories

Used to provision new repositories and grant user access.

| Method | URL             |
| :----- | :-------------- |
| `POST` | `/repositories` |

#### Request Body Structure

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

#### Request Rules for Creation

The combination of `Shared` and `AppID` defines the type of repository access being provisioned.

| Setting              | `Shared = false` (Project-Specific)                                       | `Shared = true` (Shared Access)                                 |
| :------------------- | :------------------------------------------------------------------------ | :-------------------------------------------------------------- |
| **`AppID`**          | **Required** (e.g., `"my-app-001"`)                                       | **Must be empty** (`""`)                                        |
| **`PackageManager`** | **Required** (e.g., `"npm"`, `"maven"`)                                   | **Required** (e.g., `"npm"`, `"maven"`)                         |
| **Effect**           | Creates a dedicated repository and role for a specific project (`AppID`). | Creates or assigns the user the general shared repository role. |

---

### 2. Delete Repositories

Used to remove repositories and revoke user access.

| Method   | URL             |
| :------- | :-------------- |
| `DELETE` | `/repositories` |

#### Request Body Example (Standard Deletion)

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

#### Request Rules for Deletion (Validation is Strict)

The deletion endpoint supports two distinct modes.

**Mode A: Deleting a Specific App Repository**

| Field            | Requirement  | Effect                                                                   |
| :--------------- | :----------- | :----------------------------------------------------------------------- |
| `Shared`         | `false`      | Deletes the specific repository and cleans up the associated user roles. |
| `AppID`          | **Required** |                                                                          |
| `PackageManager` | **Required** |                                                                          |

**Mode B: 💀 Offboarding / Full App Cleanup** (Shared Access & Full Teardown)

This is an "Offboarding Mode" that performs a wider cleanup based on the `AppID`.

| Field            | Requirement              | Effect                                                                                                                                                   |
| :--------------- | :----------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Shared`         | `true`                   | Disables the user, removes their named role, and deletes **all** repositories/privileges ending in `-release-<AppID>` regardless of the package manager. |
| `AppID`          | **Required**             |                                                                                                                                                          |
| `PackageManager` | **Must be empty** (`""`) |                                                                                                                                                          |

> **IQ Server impact:** Offboarding automatically revokes the Owner role in the IQ Server organization mapped to `OrganizationName`, so the user loses organization-wide Owner access along with their repositories and privileges.

> **Note:** The API rejects `DELETE` requests where `Shared=true` and `AppID` is empty. Use **Mode B** (with an `AppID`) to remove shared access from a user.

---

### 3. Check Job Status

Used to retrieve the current status and metrics of an asynchronous job.

| Method | URL              |
| :----- | :--------------- |
| `GET`  | `/jobs/<JOB_ID>` |

#### Response States & Property Case

| Property | Details                                                                                     |
| :------- | :------------------------------------------------------------------------------------------ |
| `status` | One of: `pending`, `processing`, `completed`, `failed`.                                     |
| `action` | The operation type: `create` or `delete`.                                                   |
| **Case** | **Response properties are always `camelCase`** (e.g., `successfulOperations`, `createdAt`). |

#### Example Response (When Done)

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

---

## ⚙️ Key Constraints & Data Rules

- **Field Names are Case-Sensitive in the Request:** Always use the exact casing specified in the documentation (e.g., `OrganizationName`, `LdapUsername`, `AppID`).
- **OrganizationName:** Must match exactly (case-sensitive) a key configured in the system's `config/organizations.json`.
- **PackageManager:** Must be a supported type (e.g., `npm`, `maven`, `docker`) and exist in `config/packageManager.json`.

---

## 🚨 Common API Errors

| HTTP Code | Error Message          | Common Cause                                                                                                                 |
| :-------- | :--------------------- | :--------------------------------------------------------------------------------------------------------------------------- |
| **401**   | `Unauthorized`         | Missing or incorrect `Authorization: Bearer` token.                                                                          |
| **422**   | `Unprocessable Entity` | Request JSON is malformed, or a logic rule was violated (e.g., sending `PackageManager` during a Shared Delete/Offboarding). |
| **404**   | `Not Found`            | The requested Job ID does not exist. (Jobs are in-memory and may be lost if the server restarts).                            |
