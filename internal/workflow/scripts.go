package workflow

import (
	"bytes"
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
)

//go:embed run.sh
var runScript string

//go:embed comment.sh
var commentScript string

//go:embed merge.sh
var mergeScript string

// scriptData holds the template variables injected into each .sh file.
type scriptData struct {
	BotName     string
	BotEmail    string
	DryRunFlag  string // empty or " --dry-run"
	VerboseFlag string // empty or " --verbose"
}

func newScriptData(dryRun, verbose bool) scriptData {
	dryRunFlag := ""
	if dryRun {
		dryRunFlag = " --dry-run"
	}
	verboseFlag := ""
	if verbose {
		verboseFlag = " --verbose"
	}
	return scriptData{
		BotName:     config.DefaultAppName + "[bot]",
		BotEmail:    config.DefaultAppName + "[bot]@users.noreply.github.com",
		DryRunFlag:  dryRunFlag,
		VerboseFlag: verboseFlag,
	}
}

func renderScript(tmplText string, data scriptData) string {
	tmpl, err := template.New("script").Parse(tmplText)
	if err != nil {
		return tmplText
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return tmplText
	}
	return buf.String()
}

// buildParameters builds workflow parameters from the params map
func buildParameters(params map[string]string) []map[string]interface{} {
	allParams := []string{"project-path", "instructions-md", "comment-body", "pr-number"}
	var parameters []map[string]interface{}
	for _, name := range allParams {
		param := map[string]interface{}{"name": name}
		if value, exists := params[name]; exists {
			param["value"] = value
		} else {
			param["value"] = ""
		}
		parameters = append(parameters, param)
	}
	return parameters
}

// buildRunScript returns the rendered run.sh script for a regular development workflow.
func buildRunScript(dryRun, verbose bool, _ *config.RalphConfig) string {
	return renderScript(runScript, newScriptData(dryRun, verbose))
}

// buildCommentScript returns the rendered comment.sh script for a comment-triggered workflow.
func buildCommentScript(dryRun, verbose bool) string {
	return renderScript(commentScript, newScriptData(dryRun, verbose))
}

// buildMergeScript returns the rendered merge.sh script for a merge workflow.
func buildMergeScript() string {
	return renderScript(mergeScript, newScriptData(false, false))
}

// buildVolumeMounts builds volume mounts for secrets and configMaps
func buildVolumeMounts(cfg *config.RalphConfig) []map[string]interface{} {
	mounts := []map[string]interface{}{
		{"name": "github-credentials", "mountPath": "/secrets/github", "readOnly": true},
		{"name": "opencode-credentials", "mountPath": "/secrets/opencode", "readOnly": true},
	}

	for i, cm := range cfg.Workflow.ConfigMaps {
		mount := map[string]interface{}{
			"name":     sanitizeName(cm.Name),
			"readOnly": true,
		}
		if cm.DestFile != "" {
			mount["mountPath"] = "/workspace/" + cm.DestFile
			mount["subPath"] = filepath.Base(cm.DestFile)
			mount["name"] = fmt.Sprintf("%s-%d", sanitizeName(cm.Name), i)
		} else if cm.DestDir != "" {
			mount["mountPath"] = "/workspace/" + cm.DestDir
		} else {
			mount["mountPath"] = fmt.Sprintf("/configmaps/%s", cm.Name)
		}
		mounts = append(mounts, mount)
	}

	for i, secret := range cfg.Workflow.Secrets {
		mount := map[string]interface{}{
			"name":     sanitizeName(secret.Name),
			"readOnly": true,
		}
		if secret.DestFile != "" {
			mount["mountPath"] = "/workspace/" + secret.DestFile
			mount["subPath"] = filepath.Base(secret.DestFile)
			mount["name"] = fmt.Sprintf("%s-%d", sanitizeName(secret.Name), i)
		} else if secret.DestDir != "" {
			mount["mountPath"] = "/workspace/" + secret.DestDir
		} else {
			mount["mountPath"] = fmt.Sprintf("/secrets/%s", secret.Name)
		}
		mounts = append(mounts, mount)
	}

	return mounts
}

// buildVolumes builds volumes for secrets and configMaps
func buildVolumes(cfg *config.RalphConfig) []map[string]interface{} {
	volumes := []map[string]interface{}{
		{
			"name": "github-credentials",
			"secret": map[string]interface{}{
				"secretName": k8s.GitHubSecretName,
			},
		},
		{
			"name": "opencode-credentials",
			"secret": map[string]interface{}{
				"secretName": k8s.OpenCodeSecretName,
			},
		},
	}

	for i, cm := range cfg.Workflow.ConfigMaps {
		volumeName := sanitizeName(cm.Name)
		if cm.DestFile != "" {
			volumeName = fmt.Sprintf("%s-%d", sanitizeName(cm.Name), i)
			volumes = append(volumes, map[string]interface{}{
				"name": volumeName,
				"configMap": map[string]interface{}{
					"name": cm.Name,
					"items": []map[string]interface{}{
						{"key": filepath.Base(cm.DestFile), "path": filepath.Base(cm.DestFile)},
					},
				},
			})
		} else {
			volumes = append(volumes, map[string]interface{}{
				"name":      volumeName,
				"configMap": map[string]interface{}{"name": cm.Name},
			})
		}
	}

	for i, secret := range cfg.Workflow.Secrets {
		volumeName := sanitizeName(secret.Name)
		if secret.DestFile != "" {
			volumeName = fmt.Sprintf("%s-%d", sanitizeName(secret.Name), i)
			volumes = append(volumes, map[string]interface{}{
				"name": volumeName,
				"secret": map[string]interface{}{
					"secretName": secret.Name,
					"items": []map[string]interface{}{
						{"key": filepath.Base(secret.DestFile), "path": filepath.Base(secret.DestFile)},
					},
				},
			})
		} else {
			volumes = append(volumes, map[string]interface{}{
				"name":   volumeName,
				"secret": map[string]interface{}{"secretName": secret.Name},
			})
		}
	}

	return volumes
}

// sanitizeName sanitizes a name for use as a Kubernetes volume/resource name.
func sanitizeName(name string) string {
	// Replace common special characters with hyphens
	sanitized := strings.ReplaceAll(name, "_", "-")
	sanitized = strings.ReplaceAll(sanitized, ".", "-")
	sanitized = strings.ReplaceAll(sanitized, "/", "-")
	sanitized = strings.ToLower(sanitized)

	// Remove any other non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range sanitized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		} else {
			result.WriteRune('-')
		}
	}

	// Remove leading/trailing hyphens and ensure not empty
	sanitized = result.String()
	sanitized = strings.Trim(sanitized, "-")
	if sanitized == "" {
		return "default"
	}

	// Ensure it starts with a letter
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "branch-" + sanitized
	}

	return sanitized
}
