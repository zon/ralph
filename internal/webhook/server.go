package webhook

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/webhookconfig"
)

// Server is the GitHub webhook HTTP server.
type Server struct {
	config     *webhookconfig.Config
	router     *gin.Engine
	out        *output.Client
	argoClient argo.Client
}

// NewServer creates a new webhook Server with the given configuration.
func NewServer(cfg *webhookconfig.Config, out *output.Client, argoClient argo.Client) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		config:     cfg,
		router:     router,
		out:        out,
		argoClient: argoClient,
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	payload, err := github.ParseWebhookPayload(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	owner, repoName := payload.RepoOwner(), payload.RepoName()
	if owner == "" || repoName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unable to identify repository from payload"})
		return
	}

	s.out.Debugf("received webhook for %s/%s", owner, repoName)

	secret := s.config.WebhookSecretForRepo(owner, repoName)
	if secret == "" {
		s.out.Debugf("ignoring event: repo %s/%s not configured", owner, repoName)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "repository not configured"})
		return
	}

	sig := c.GetHeader("X-Hub-Signature-256")
	if !webhookconfig.ValidateSignature(body, secret, sig) {
		s.out.Debugf("rejected request: invalid signature for %s/%s", owner, repoName)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	eventType := c.GetHeader("X-GitHub-Event")
	msg := payload.Review.Body
	if msg == "" {
		msg = payload.Comment.Body
	}
	s.out.Debugf("incoming %s (%s) for %s/%s PR#%d branch=%s body=%q", eventType, payload.Action, owner, repoName, payload.PullRequest.Number, payload.PullRequest.Head.Ref, msg)

	if !payload.IsAcceptable(eventType, s.config) {
		s.out.Debugf("ignoring %s event for %s/%s", eventType, owner, repoName)
		c.Status(http.StatusOK)
		return
	}

	fields := payload.ToEvent(eventType)
	event := Event{
		Body:      fields.Body,
		Approved:  fields.Approved,
		PRBranch:  fields.PRBranch,
		RepoOwner: fields.RepoOwner,
		RepoName:  fields.RepoName,
		PRNumber:  fields.PRNumber,
		Author:    fields.Author,
	}
	s.out.Debugf("dispatching %s for %s/%s", eventType, owner, repoName)

	result, err := event.ToWorkflow(s.config)
	if err != nil {
		s.out.Debugf("failed to generate workflow for %s/%s: %v", owner, repoName, err)
		c.Status(http.StatusOK)
		return
	}

	go s.submitWorkflow(result, owner, repoName)
	c.Status(http.StatusOK)
}

// submitWorkflow submits a WorkflowResult asynchronously.
func (s *Server) submitWorkflow(result *WorkflowResult, owner, repoName string) {
	ctx := context.Background()
	if result.Run != nil {
		name, err := result.Run.Submit(ctx, s.argoClient)
		if err != nil {
			s.out.Debugf("failed to submit run workflow for %s/%s: %v", owner, repoName, err)
			return
		}
		s.out.Debugf("submitted run workflow %s for %s/%s", name, owner, repoName)
		s.out.Debugf("To watch logs, run: argo logs -n %s -f %s", result.Namespace, name)
	} else if result.Merge != nil {
		name, err := result.Merge.Submit(ctx, s.argoClient)
		if err != nil {
			s.out.Debugf("failed to submit merge workflow for %s/%s: %v", owner, repoName, err)
			return
		}
		s.out.Debugf("submitted merge workflow %s for %s/%s", name, owner, repoName)
		s.out.Debugf("To watch logs, run: argo logs -n %s -f %s", result.Namespace, name)
	}
}

