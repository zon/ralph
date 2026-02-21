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
)

// EventHandler is called when a webhook event passes all filters.
// eventType is the X-GitHub-Event header value.
// repoOwner and repoName identify the repository the event belongs to.
// payload is the parsed JSON payload.
type EventHandler func(eventType string, repoOwner, repoName string, payload map[string]interface{})

// Server is the GitHub webhook HTTP server.
type Server struct {
	config  *Config
	handler EventHandler
	router  *gin.Engine
}

// NewServer creates a new webhook Server with the given configuration and event handler.
// The handler is called for events that pass all validation and filtering checks.
// Pass a nil handler if no processing is needed (useful for testing).
func NewServer(cfg *Config, handler EventHandler) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		config:  cfg,
		handler: handler,
		router:  router,
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
func (s *Server) handleWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// Identify the repository from the payload before we can look up the
	// per-repo webhook secret. We parse just enough to get owner/name.
	owner, repoName, err := extractRepoFromPayload(body)
	if err != nil || owner == "" || repoName == "" {
		// Cannot identify repository – reject with 400.
		c.JSON(http.StatusBadRequest, gin.H{"error": "unable to identify repository from payload"})
		return
	}

	// Look up the webhook secret for this repository.
	secret := s.config.WebhookSecretForRepo(owner, repoName)
	if secret == "" {
		// Repository not configured – reject.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "repository not configured"})
		return
	}

	// Validate the HMAC-SHA256 signature.
	sig := c.GetHeader("X-Hub-Signature-256")
	if !validateSignature(body, secret, sig) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	eventType := c.GetHeader("X-GitHub-Event")

	// issue_comment events are explicitly ignored.
	if eventType == "issue_comment" {
		c.Status(http.StatusOK)
		return
	}

	// Parse the full payload for event-type-specific filtering.
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	ralphUsername := s.config.App.RalphUsername

	switch eventType {
	case "pull_request_review_comment":
		// Only process events on PRs opened by ralph.
		prAuthor, _ := nestedString(payload, "pull_request", "user", "login")
		if !strings.EqualFold(prAuthor, ralphUsername) {
			c.Status(http.StatusOK)
			return
		}
		// Ignore events posted by ralph.
		commenter, _ := nestedString(payload, "comment", "user", "login")
		if strings.EqualFold(commenter, ralphUsername) {
			c.Status(http.StatusOK)
			return
		}
		if s.handler != nil {
			s.handler(eventType, owner, repoName, payload)
		}
		c.Status(http.StatusOK)

	case "pull_request_review":
		// Only process reviews where state == "approved".
		state, _ := nestedString(payload, "review", "state")
		if strings.ToLower(state) != "approved" {
			c.Status(http.StatusOK)
			return
		}
		// Only process events on PRs opened by ralph.
		prAuthor, _ := nestedString(payload, "pull_request", "user", "login")
		if !strings.EqualFold(prAuthor, ralphUsername) {
			c.Status(http.StatusOK)
			return
		}
		// Ignore reviews posted by ralph.
		reviewer, _ := nestedString(payload, "review", "user", "login")
		if strings.EqualFold(reviewer, ralphUsername) {
			c.Status(http.StatusOK)
			return
		}
		if s.handler != nil {
			s.handler(eventType, owner, repoName, payload)
		}
		c.Status(http.StatusOK)

	default:
		// Unrecognised event types are ignored gracefully.
		c.Status(http.StatusOK)
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

// extractRepoFromPayload parses just enough of the JSON payload to return the
// repository owner and name.
func extractRepoFromPayload(body []byte) (owner, name string, err error) {
	var partial struct {
		Repository struct {
			Name  string `json:"name"`
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &partial); err != nil {
		return "", "", err
	}
	return partial.Repository.Owner.Login, partial.Repository.Name, nil
}

// nestedString is a helper that walks a map hierarchy using keys and returns the
// final string value. Returns "" if any key is missing or the final value is not a string.
func nestedString(m map[string]interface{}, keys ...string) (string, bool) {
	var current interface{} = m
	for _, k := range keys {
		mp, ok := current.(map[string]interface{})
		if !ok {
			return "", false
		}
		current = mp[k]
	}
	s, ok := current.(string)
	return s, ok
}
