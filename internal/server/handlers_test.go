package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter(bm *BatchManager) (*gin.Engine, *Handler) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	cfg := &config.Config{
		APIToken: "test-token",
		Orgs: map[string]string{
			"org1": "org-id-1",
		},
		PackageManagers: map[string]config.PackageManager{
			"npm": {DefaultURL: "https://registry.npmjs.org"},
		},
	}
	jobStore := config.NewJobStore()
	handler := newHandler(cfg, jobStore, bm)

	return r, handler
}

func TestHealth(t *testing.T) {
	r, h := setupRouter(nil)
	r.GET("/health", h.health)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])
	assert.Equal(t, "healthy", resp["status"])
}

func TestAuthMiddleware(t *testing.T) {
	r, _ := setupRouter(nil)
	r.Use(authMiddleware("test-token"))
	r.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("Authorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Unauthorized - Wrong Token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer wrong-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Unauthorized - No Header", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestGetJobStatus(t *testing.T) {
	r, h := setupRouter(nil)
	r.GET("/jobs/:id", h.getJobStatus)

	// Pre-populate a job
	h.jobStore.CreateJob("job-1", "create", 1)

	t.Run("Job Found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/jobs/job-1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "job-1", resp["id"])
	})

	t.Run("Job Not Found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/jobs/job-999", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleBatch_Validation(t *testing.T) {
	r, h := setupRouter(nil)
	r.POST("/batch", h.createBatch)

	t.Run("Empty Body", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/batch", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Empty Requests List", func(t *testing.T) {
		body := batchRepositoryRequest{Requests: []config.RepositoryRequest{}}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/batch", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

func TestCreateBatch_Success(t *testing.T) {
	mockNexus := new(MockNexusClient)
	mockIQ := new(MockIQClient)
	cfg := &config.Config{
		Orgs: map[string]string{"org1": "org-id-1"},
		PackageManagers: map[string]config.PackageManager{
			"npm": {DefaultURL: "https://registry.npmjs.org"},
		},
	}
	jobStore := config.NewJobStore()
	bm := NewBatchManager(cfg, jobStore, mockNexus, mockIQ)

	r, h := setupRouter(bm)
	r.POST("/batch", h.createBatch)

	reqBody := batchRepositoryRequest{
		Requests: []config.RepositoryRequest{
			{
				OrganizationName: "org1",
				PackageManager:   "npm",
				AppID:            "app1",
				LdapUsername:     "user1",
				Shared:           false,
			},
		},
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/batch", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	// We expect ProcessBatchAsync to be called.
	// Since it runs in a goroutine, we can't easily mock the async part's effect on the client immediately
	// unless we wait, but the handler returns immediately.
	// The handler just calls ProcessBatchAsync which queues the job.
	// We don't need to mock Nexus/IQ calls here because they happen in the background goroutine.
	// However, if ProcessBatchAsync does any sync work that calls clients, we'd need to mock it.
	// Looking at ProcessBatchAsync, it only creates a job in store and starts a goroutine.
	// So no client calls on the main thread.

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp AcceptedResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.JobID)
	assert.Equal(t, StatusPending, resp.Status)
}

func TestDeleteBatch_Success(t *testing.T) {
	mockNexus := new(MockNexusClient)
	mockIQ := new(MockIQClient)
	cfg := &config.Config{
		Orgs: map[string]string{"org1": "org-id-1"},
		PackageManagers: map[string]config.PackageManager{
			"npm": {DefaultURL: "https://registry.npmjs.org"},
		},
	}
	jobStore := config.NewJobStore()
	bm := NewBatchManager(cfg, jobStore, mockNexus, mockIQ)

	r, h := setupRouter(bm)
	r.DELETE("/batch", h.deleteBatch)

	reqBody := batchRepositoryRequest{
		Requests: []config.RepositoryRequest{
			{
				OrganizationName: "org1",
				PackageManager:   "npm",
				AppID:            "app1",
				LdapUsername:     "user1",
				Shared:           false,
			},
		},
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("DELETE", "/batch", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
}
