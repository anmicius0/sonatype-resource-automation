package server

import (
	"github.com/anmicius0/sonatype-resource-automation/internal/config"
	"github.com/gin-gonic/gin"
)

// NewRouter builds the Gin router with the configured API handlers.
func NewRouter(cfg *config.Config, jobStore *config.JobStore, batchManager *BatchManager) *gin.Engine {
	router := gin.Default()
	router.Use(gin.Logger())

	handler := newHandler(cfg, jobStore, batchManager)

	router.GET(HealthEndpoint, handler.health)
	router.POST(RepositoriesPath, authMiddleware(cfg.APIToken), handler.createBatch)
	router.DELETE(RepositoriesPath, authMiddleware(cfg.APIToken), handler.deleteBatch)
	router.GET(JobsPath+"/:id", authMiddleware(cfg.APIToken), handler.getJobStatus)

	return router
}
