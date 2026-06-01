package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

func RegisterWebhook(ctx context.Context, owner, repo, webhookURL, secret string) error {
	listCmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/%s/hooks", owner, repo),
		"--jq", fmt.Sprintf(`.[] | select(.config.url | contains("%s")) | .id`, webhookURL),
	)
	var listOut, listErr bytes.Buffer
	listCmd.Stdout = &listOut
	listCmd.Stderr = &listErr

	var existingID string
	if err := listCmd.Run(); err == nil {
		existingID = strings.TrimSpace(listOut.String())
	}

	payload := map[string]interface{}{
		"name":   "web",
		"active": true,
		"events": []string{"push", "pull_request", "pull_request_review", "issue_comment"},
		"config": map[string]string{
			"url":          webhookURL,
			"content_type": "json",
			"secret":       secret,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	if existingID != "" && existingID != "null" {
		updateCmd := exec.CommandContext(ctx, "gh", "api",
			fmt.Sprintf("repos/%s/%s/hooks/%s", owner, repo, existingID),
			"--method", "PATCH",
			"--input", "-",
		)
		updateCmd.Stdin = bytes.NewReader(payloadBytes)
		var updateErr bytes.Buffer
		updateCmd.Stderr = &updateErr
		if err := updateCmd.Run(); err != nil {
			return fmt.Errorf("failed to update webhook for %s/%s: %w (stderr: %s)",
				owner, repo, err, updateErr.String())
		}
		return nil
	}

	createCmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/%s/hooks", owner, repo),
		"--method", "POST",
		"--input", "-",
	)
	createCmd.Stdin = bytes.NewReader(payloadBytes)
	var createErr bytes.Buffer
	createCmd.Stderr = &createErr
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create webhook for %s/%s: %w (stderr: %s)",
			owner, repo, err, createErr.String())
	}

	return nil
}
