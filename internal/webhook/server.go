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

	logger.Verbosef("received webhook for %s/%s", owner, repoName)

	// Look up the webhook secret for this repository.
	secret := s.config.WebhookSecretForRepo(owner, repoName)
	if secret == "" {
		// Repository not configured – reject.
		logger.Verbosef("ignoring event: repo %s/%s not configured", owner, repoName)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "repository not configured"})
		return
	}

	// Validate the HMAC-SHA256 signature.
	sig := c.GetHeader("X-Hub-Signature-256")
	if !validateSignature(body, secret, sig) {
		logger.Verbosef("rejected request: invalid signature for %s/%s", owner, repoName)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	eventType := c.GetHeader("X-GitHub-Event")

	// Parse the full payload for event-type-specific filtering.
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	repo := s.config.RepoByFullName(owner, repoName)

	switch eventType {
	case "issue_comment":
		// Only process issue_comment events that are on a pull request.
		// GitHub includes an "issue.pull_request" field when the comment is on a PR.
		_, isPR := nestedString(payload, "issue", "pull_request", "url")
		if !isPR {
			logger.Verbosef("ignoring issue_comment event on non-PR issue for %s/%s", owner, repoName)
			c.Status(http.StatusOK)
			return
		}
		commenter, _ := nestedString(payload, "comment", "user", "login")
		if s.config.IsUserIgnored(repo, commenter) {
			logger.Verbosef("ignoring issue_comment: user %q is in ignored-users list for %s/%s", commenter, owner, repoName)
			c.Status(http.StatusOK)
			return
		}
		if repo != nil && !repo.IsUserAllowed(commenter) {
			logger.Verbosef("ignoring issue_comment: user %q not in allowlist for %s/%s", commenter, owner, repoName)
			c.Status(http.StatusOK)
			return
		}
		logger.Verbosef("dispatching issue_comment (on PR) for %s/%s (user: %s)", owner, repoName, commenter)
		if s.handler != nil {
			s.handler(eventType, owner, repoName, payload)
		}
		c.Status(http.StatusOK)

	case "pull_request_review_comment":
		commenter, _ := nestedString(payload, "comment", "user", "login")
		// Ignore events from ignored users.
		if s.config.IsUserIgnored(repo, commenter) {
			logger.Verbosef("ignoring pull_request_review_comment: user %q is in ignored-users list for %s/%s", commenter, owner, repoName)
			c.Status(http.StatusOK)
			return
		}
		// Only process events from allowed users.
		if repo != nil && !repo.IsUserAllowed(commenter) {
			logger.Verbosef("ignoring pull_request_review_comment: user %q not in allowlist for %s/%s", commenter, owner, repoName)
			c.Status(http.StatusOK)
			return
		}
		logger.Verbosef("dispatching pull_request_review_comment for %s/%s (user: %s)", owner, repoName, commenter)
		if s.handler != nil {
			s.handler(eventType, owner, repoName, payload)
		}
		c.Status(http.StatusOK)

	case "pull_request_review":
		// Only process reviews where state == "approved".
		state, _ := nestedString(payload, "review", "state")
		if strings.ToLower(state) != "approved" {
			logger.Verbosef("ignoring pull_request_review: state is %q (not approved) for %s/%s", state, owner, repoName)
			c.Status(http.StatusOK)
			return
		}
		reviewer, _ := nestedString(payload, "review", "user", "login")
		// Ignore events from ignored users.
		if s.config.IsUserIgnored(repo, reviewer) {
			logger.Verbosef("ignoring pull_request_review: user %q is in ignored-users list for %s/%s", reviewer, owner, repoName)
			c.Status(http.StatusOK)
			return
		}
		// Only process reviews from allowed users.
		if repo != nil && !repo.IsUserAllowed(reviewer) {
			logger.Verbosef("ignoring pull_request_review: user %q not in allowlist for %s/%s", reviewer, owner, repoName)
			c.Status(http.StatusOK)
			return
		}
		logger.Verbosef("dispatching pull_request_review (approved) for %s/%s (reviewer: %s)", owner, repoName, reviewer)
		if s.handler != nil {
			s.handler(eventType, owner, repoName, payload)
		}
		c.Status(http.StatusOK)

	default:
		// Unrecognised event types are ignored gracefully.
		logger.Verbosef("ignoring unrecognised event type %q for %s/%s", eventType, owner, repoName)
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
