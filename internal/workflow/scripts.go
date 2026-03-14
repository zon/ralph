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
	VerboseFlag string // empty or " --verbose"
	DebugBranch string // empty or the ralph repo branch to use for go run mode
}

func newScriptData(verbose bool, debugBranch string) scriptData {
	verboseFlag := ""
	if verbose {
		verboseFlag = " --verbose"
	}
	return scriptData{
		BotName:     config.DefaultAppName + "[bot]",
		BotEmail:    config.DefaultAppName + "[bot]@users.noreply.github.com",
		VerboseFlag: verboseFlag,
		DebugBranch: debugBranch,
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
	allParams := []string{"project-path", "instructions-md", "comment-body", "pr-number", "base-branch"}
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
func buildRunScript(verbose bool, debugBranch string, _ *config.RalphConfig) string {
	return renderScript(runScript, newScriptData(verbose, debugBranch))
}

// buildCommentScript returns the rendered comment.sh script for a comment-triggered workflow.
func buildCommentScript(verbose bool) string {
	return renderScript(commentScript, newScriptData(verbose, ""))
}

// buildMergeScript returns the rendered merge.sh script for a merge workflow.
func buildMergeScript() string {
	return renderScript(mergeScript, newScriptData(false, ""))
}

func buildConfigMapVolumeMount(name string, destFile, destDir string, index int) map[string]interface{} {
	mount := map[string]interface{}{
		"name":     sanitizeName(name),
		"readOnly": true,
	}
	if destFile != "" {
		mount["mountPath"] = "/workspace/" + destFile
		mount["subPath"] = filepath.Base(destFile)
		mount["name"] = fmt.Sprintf("%s-%d", sanitizeName(name), index)
	} else if destDir != "" {
		mount["mountPath"] = "/workspace/" + destDir
	} else {
		mount["mountPath"] = fmt.Sprintf("/configmaps/%s", name)
	}
	return mount
}

func buildSecretVolumeMount(name string, destFile, destDir string, index int) map[string]interface{} {
	mount := map[string]interface{}{
		"name":     sanitizeName(name),
		"readOnly": true,
	}
	if destFile != "" {
		mount["mountPath"] = "/workspace/" + destFile
		mount["subPath"] = filepath.Base(destFile)
		mount["name"] = fmt.Sprintf("%s-%d", sanitizeName(name), index)
	} else if destDir != "" {
		mount["mountPath"] = "/workspace/" + destDir
	} else {
		mount["mountPath"] = fmt.Sprintf("/secrets/%s", name)
	}
	return mount
}

func buildConfigMapVolume(name string, destFile string, index int) map[string]interface{} {
	volumeName := sanitizeName(name)
	if destFile != "" {
		volumeName = fmt.Sprintf("%s-%d", sanitizeName(name), index)
		return map[string]interface{}{
			"name": volumeName,
			"configMap": map[string]interface{}{
				"name": name,
				"items": []map[string]interface{}{
					{"key": filepath.Base(destFile), "path": filepath.Base(destFile)},
				},
			},
		}
	}
	return map[string]interface{}{
		"name":      volumeName,
		"configMap": map[string]interface{}{"name": name},
	}
}

func buildSecretVolume(name string, destFile string, index int) map[string]interface{} {
	volumeName := sanitizeName(name)
	if destFile != "" {
		volumeName = fmt.Sprintf("%s-%d", sanitizeName(name), index)
		return map[string]interface{}{
			"name": volumeName,
			"secret": map[string]interface{}{
				"secretName": name,
				"items": []map[string]interface{}{
					{"key": filepath.Base(destFile), "path": filepath.Base(destFile)},
				},
			},
		}
	}
	return map[string]interface{}{
		"name":   volumeName,
		"secret": map[string]interface{}{"secretName": name},
	}
}

func buildCredentialVolumes() []map[string]interface{} {
	return []map[string]interface{}{
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
		{
			"name": "pulumi-credentials",
			"secret": map[string]interface{}{
				"secretName": k8s.PulumiSecretName,
				"optional":   true,
			},
		},
	}
}

func buildCredentialMounts() []map[string]interface{} {
	return []map[string]interface{}{
		{"name": "github-credentials", "mountPath": "/secrets/github", "readOnly": true},
		{"name": "opencode-credentials", "mountPath": "/secrets/opencode", "readOnly": true},
		{"name": "pulumi-credentials", "mountPath": "/secrets/pulumi", "readOnly": true},
	}
}

func buildVolumeMounts(cfg *config.RalphConfig) []map[string]interface{} {
	mounts := buildCredentialMounts()

	for i, cm := range cfg.Workflow.ConfigMaps {
		mounts = append(mounts, buildConfigMapVolumeMount(cm.Name, cm.DestFile, cm.DestDir, i))
	}

	for i, secret := range cfg.Workflow.Secrets {
		mounts = append(mounts, buildSecretVolumeMount(secret.Name, secret.DestFile, secret.DestDir, i))
	}

	return mounts
}

func buildVolumes(cfg *config.RalphConfig) []map[string]interface{} {
	volumes := buildCredentialVolumes()

	for i, cm := range cfg.Workflow.ConfigMaps {
		volumes = append(volumes, buildConfigMapVolume(cm.Name, cm.DestFile, i))
	}

	for i, secret := range cfg.Workflow.Secrets {
		volumes = append(volumes, buildSecretVolume(secret.Name, secret.DestFile, i))
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
