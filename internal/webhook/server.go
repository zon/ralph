package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zon/ralph/internal/logger"
)

// Server is the GitHub webhook HTTP server.
type Server struct {
	config *Config
	router *gin.Engine
}

// NewServer creates a new webhook Server with the given configuration.
func NewServer(cfg *Config) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		config: cfg,
		router: router,
	}

	router.POST("/webhook", s.handleWebhook)

	return s
}

// Router returns the underlying gin.Engine so it can be used directly in tests
// without starting a real HTTP listener.
func (s *Server) Router() http.Handler {
	return s.router
}

// Run starts the HTTP server on the configured port. It blocks until the server
// encounters a fatal error.
func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.config.App.Port)
	return s.router.Run(addr)
}

// handleWebhook is the main Gin handler for POST /webhook.
// It runs the full pipeline: receive → validate → filter → event → workflow → submit.
func (s *Server) handleWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	var payload GithubPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	owner, repoName := payload.RepoOwner(), payload.RepoName()
	if owner == "" || repoName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unable to identify repository from payload"})
		return
	}

	logger.Verbosef("received webhook for %s/%s", owner, repoName)

	secret := s.config.WebhookSecretForRepo(owner, repoName)
	if secret == "" {
		logger.Verbosef("ignoring event: repo %s/%s not configured", owner, repoName)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "repository not configured"})
		return
	}

	sig := c.GetHeader("X-Hub-Signature-256")
	if !validateSignature(body, secret, sig) {
		logger.Verbosef("rejected request: invalid signature for %s/%s", owner, repoName)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	eventType := c.GetHeader("X-GitHub-Event")
	msg := payload.Review.Body
	if msg == "" {
		msg = payload.Comment.Body
	}
	logger.Verbosef("incoming %s (%s) for %s/%s PR#%d branch=%s body=%q", eventType, payload.Action, owner, repoName, payload.PullRequest.Number, payload.PullRequest.Head.Ref, msg)

	if !payload.IsAcceptable(eventType, s.config) {
		logger.Verbosef("ignoring %s event for %s/%s", eventType, owner, repoName)
		c.Status(http.StatusOK)
		return
	}

	event := payload.ToEvent(eventType)
	logger.Verbosef("dispatching %s for %s/%s", eventType, owner, repoName)

	result, err := event.ToWorkflow(s.config)
	if err != nil {
		logger.Verbosef("failed to generate workflow for %s/%s: %v", owner, repoName, err)
		c.Status(http.StatusOK)
		return
	}

	go submitWorkflow(result, owner, repoName)
	c.Status(http.StatusOK)
}

// submitWorkflow submits a WorkflowResult asynchronously.
func submitWorkflow(result *WorkflowResult, owner, repoName string) {
	if result.Run != nil {
		name, err := result.Run.Submit(result.Namespace)
		if err != nil {
			logger.Verbosef("failed to submit run workflow for %s/%s: %v", owner, repoName, err)
			return
		}
		logger.Verbosef("submitted run workflow %s for %s/%s", name, owner, repoName)
	} else if result.Merge != nil {
		name, err := result.Merge.Submit(result.Namespace)
		if err != nil {
			logger.Verbosef("failed to submit merge workflow for %s/%s: %v", owner, repoName, err)
			return
		}
		logger.Verbosef("submitted merge workflow %s for %s/%s", name, owner, repoName)
	}
}

// validateSignature checks the X-Hub-Signature-256 header against the HMAC-SHA256
// of body using secret. Returns false if the signature is missing or invalid.
func validateSignature(body []byte, secret, signature string) bool {
	if signature == "" {
		return false
	}
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return false
	}
	sigHex := strings.TrimPrefix(signature, prefix)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sigHex))
}
