# API 使用手冊

## 如何使用

1. **發送請求**：告訴 API 你要建立或刪除什麼。
2. **取得 Job ID**：API 會回覆「收到，處理中」，並給你一個 ID。
3. **查詢進度**：用這個 ID 來檢查工作是否完成。

## 身分驗證

你需要一組 Token 才能使用。請放在 Header 裡：
`Authorization: Bearer <YOUR_API_TOKEN>`

## API 功能

### 1. 建立儲存庫

**網址**: `POST /repositories`

**內容範例**:

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

### 2. 刪除儲存庫

注意：範例 JSON 使用了與 Go struct 對應的正規化欄位名稱（首字母大寫）。解碼器對大小寫有一定容忍度，但建議使用範例中的欄位名稱，以避免混淆。
**網址**: `DELETE /repositories`

**內容範例**: 同上。

### 3. 查詢進度

**網址**: `GET /jobs/<JOB_ID>`

**回應範例 (完成時):**

`GET /jobs/<JOB_ID>` 會回傳完整 job 物件及統計資料。範例：

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

注意：回傳的欄位名稱使用 camelCase（例如：`jobId`、`createdAt`、`successfulOperations`）。

補充資訊：

- `status`：可能是 `pending`、`processing`、`completed`、`failed`。
- `action`：`create` 或 `delete`，表示工作執行的動作。

## 重要規則

- **OrganizationName**: 必須跟系統設定完全一樣 (與 `config/organizations.json` 的 key 進行精確對照)。
- **PackageManager**: 例如 `npm`, `maven`, `docker` — 必須存在於 `config/packageManager.json`。
- **Shared (共用設定)**:
  - 如果是 `true` (共用)：`AppID` 必須留空。
  - 如果是 `false` (專案專用)：一定要填寫 `AppID`。
  - 注意：`Requests` 項目中 `OrganizationName`、`LdapUsername`、`PackageManager` 為必要欄位。伺服器會檢查 `Shared` 與 `AppID` 的組合，若不合法會回傳 `422` 錯誤。

## 常見錯誤

- **401 Unauthorized**: Token 錯了或沒帶。
- **422 Unprocessable Entity**: 資料填錯了 (例如漏填欄位)。
- **404 Not Found**: 找不到這個 Job ID (可能是伺服器重啟過)。
