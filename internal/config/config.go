package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed development-instructions.md
var defaultInstructions string

//go:embed comment-instructions.md
var defaultCommentInstructions string

//go:embed merge-instructions.md
var defaultMergeInstructions string

//go:embed fix-service-instructions.md
var defaultFixServiceInstructions string

//go:embed pick-requirement-instructions.md
var defaultPickInstructions string

// Before represents a command to run before starting services
type Before struct {
	Name     string   `yaml:"name"`
	Command  string   `yaml:"command"`
	Args     []string `yaml:"args,omitempty"`
	WorkDir  string   `yaml:"workDir,omitempty"`
	Optional bool     `yaml:"optional,omitempty"`
}

// Service represents a service to be started/stopped
type Service struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args,omitempty"`
	Port    int      `yaml:"port,omitempty"`    // Optional, for health checking
	Timeout int      `yaml:"timeout,omitempty"` // Optional, health check timeout in seconds (default: 30)
	WorkDir string   `yaml:"workDir,omitempty"` // Optional, working directory for the command
}

const (
	DefaultAppName = "ralph-zon"
	DefaultAppID   = "2966665"
)

// AppInfo holds the GitHub App identity
type AppInfo struct {
	Name string `yaml:"name,omitempty"`
	ID   string `yaml:"id,omitempty"`
}

// ImageConfig represents container image configuration
type ImageConfig struct {
	Repository string `yaml:"repository,omitempty"`
	Tag        string `yaml:"tag,omitempty"`
}

// ConfigMapMount represents a ConfigMap to mount with destination info
type ConfigMapMount struct {
	Name     string `yaml:"name"`               // Name of the ConfigMap
	DestFile string `yaml:"destFile,omitempty"` // Destination file path (if mounting a single file)
	DestDir  string `yaml:"destDir,omitempty"`  // Destination directory (if mounting entire ConfigMap)
	Link     bool   `yaml:"link,omitempty"`     // Whether to create a symlink in workspace (default: false)
}

// SecretMount represents a Secret to mount with destination info
type SecretMount struct {
	Name     string `yaml:"name"`               // Name of the Secret
	DestFile string `yaml:"destFile,omitempty"` // Destination file path (if mounting a single file)
	DestDir  string `yaml:"destDir,omitempty"`  // Destination directory (if mounting entire Secret)
	Link     bool   `yaml:"link,omitempty"`     // Whether to create a symlink in workspace (default: false)
}

// WorkflowConfig represents Argo Workflow configuration options
type WorkflowConfig struct {
	Image      ImageConfig       `yaml:"image,omitempty"`
	ConfigMaps []ConfigMapMount  `yaml:"configMaps,omitempty"`
	Secrets    []SecretMount     `yaml:"secrets,omitempty"`
	Env        map[string]string `yaml:"env,omitempty"`
	Context    string            `yaml:"context,omitempty"`
	Namespace  string            `yaml:"namespace,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty"`
}

const LoopTypeDomainFunction = "domain-function"

var validLoopTypes = map[string]bool{
	LoopTypeDomainFunction: true,
}

// ReviewItem represents a single review item with exactly one source (Text, File, or URL)
type ReviewItem struct {
	Text string `yaml:"text,omitempty"` // Inline string content
	File string `yaml:"file,omitempty"` // Path relative to repo root, read at runtime
	URL  string `yaml:"url,omitempty"`  // HTTP URL fetched at runtime, expects plain text response
	Loop string `yaml:"loop,omitempty"` // Optional loop type for iterative prompting
}

// ReviewConfig represents the review configuration section
type ReviewConfig struct {
	Model string       `yaml:"model,omitempty"` // AI model to use for review
	Items []ReviewItem `yaml:"items"`           // Required list of review items
}

// RalphConfig represents the .ralph/config.yaml structure
type RalphConfig struct {
	MaxIterations       int            `yaml:"maxIterations,omitempty"`
	DefaultBranch       string         `yaml:"defaultBranch,omitempty"`
	Model               string         `yaml:"model,omitempty"` // AI model to use for coding and PR summary (default: deepseek/deepseek-chat)
	Before              []Before       `yaml:"before,omitempty"`
	Services            []Service      `yaml:"services,omitempty"`
	Workflow            WorkflowConfig `yaml:"workflow,omitempty"`
	App                 AppInfo        `yaml:"app,omitempty"`
	Review              ReviewConfig   `yaml:"review,omitempty"`
	ConfigPath          string         `yaml:"-"` // Path to the loaded config file
	Instructions        string         `yaml:"-"` // Not persisted in YAML, loaded from .ralph/instructions.md
	CommentInstructions string         `yaml:"-"` // Not persisted in YAML, loaded from .ralph/comment-instructions.md
	MergeInstructions   string         `yaml:"-"` // Not persisted in YAML, loaded from .ralph/merge-instructions.md
}

func (c *RalphConfig) DefaultCommentInstructions() string {
	return DefaultCommentInstructions()
}

func (c *RalphConfig) DefaultMergeInstructions() string {
	return DefaultMergeInstructions()
}

func DefaultCommentInstructions() string {
	return defaultCommentInstructions
}

func DefaultMergeInstructions() string {
	return defaultMergeInstructions
}

func DefaultDevelopmentInstructions() string {
	return defaultInstructions
}

func DefaultFixServiceInstructions() string {
	return defaultFixServiceInstructions
}

func DefaultPickInstructions() string {
	return defaultPickInstructions
}

// ValidateReviewConfig validates the review configuration
func ValidateReviewConfig(r *ReviewConfig) error {
	if len(r.Items) == 0 {
		return fmt.Errorf("review must have at least one item")
	}

	for i, item := range r.Items {
		count := 0
		if item.Text != "" {
			count++
		}
		if item.File != "" {
			count++
		}
		if item.URL != "" {
			count++
		}

		if count == 0 {
			return fmt.Errorf("review item %d must have one of text, file, or url set", i)
		}
		if count > 1 {
			return fmt.Errorf("review item %d must have exactly one of text, file, or url set", i)
		}

		if item.Loop != "" && !validLoopTypes[item.Loop] {
			return fmt.Errorf("review item %d has invalid loop type %q; valid types are: domain-function", i, item.Loop)
		}
	}

	return nil
}

// applyDefaults fills in zero-value fields with their default values.
func applyDefaults(config *RalphConfig) {
	if config.MaxIterations == 0 {
		config.MaxIterations = 10
	}
	if config.DefaultBranch == "" {
		config.DefaultBranch = "main"
	}
	if config.Model == "" {
		config.Model = "deepseek/deepseek-chat"
	}
	if config.App.Name == "" {
		config.App.Name = DefaultAppName
	}
	if config.App.ID == "" {
		config.App.ID = DefaultAppID
	}

	for i := range config.Services {
		if config.Services[i].Timeout == 0 {
			config.Services[i].Timeout = 30
		}
	}
}

// FindConfigDir searches upwards from startDir for a .ralph directory
func FindConfigDir(startDir string) (string, error) {
	curr := startDir
	for {
		configDir := filepath.Join(curr, ".ralph")
		if info, err := os.Stat(configDir); err == nil && info.IsDir() {
			return configDir, nil
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			return "", os.ErrNotExist
		}
		curr = parent
	}
}

// loadConfigFromPath reads and parses the config file at the given path.
// If the file does not exist, it returns an empty config and nil error.
// If the file exists but cannot be read or parsed, it returns an error.
func loadConfigFromPath(configPath string) (*RalphConfig, error) {
	var config RalphConfig
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Missing config file is allowed (use zero values)
			return &config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}
	config.ConfigPath = configPath
	return &config, nil
}

// loadInstructions loads the instruction files from the config directory.
// If a file does not exist, the corresponding default instructions are used.
func loadInstructions(configDir string) (instructions, commentInstructions, mergeInstructions string) {
	// Load instructions from .ralph/instructions.md or use default
	instructionsPath := filepath.Join(configDir, "instructions.md")
	if instructionsData, err := os.ReadFile(instructionsPath); err == nil {
		instructions = string(instructionsData)
	} else {
		instructions = defaultInstructions
	}

	// Load comment instructions from .ralph/comment-instructions.md or use default
	commentInstructionsPath := filepath.Join(configDir, "comment-instructions.md")
	if data, err := os.ReadFile(commentInstructionsPath); err == nil {
		commentInstructions = string(data)
	} else {
		commentInstructions = defaultCommentInstructions
	}

	// Load merge instructions from .ralph/merge-instructions.md or use default
	mergeInstructionsPath := filepath.Join(configDir, "merge-instructions.md")
	if data, err := os.ReadFile(mergeInstructionsPath); err == nil {
		mergeInstructions = string(data)
	} else {
		mergeInstructions = defaultMergeInstructions
	}

	return
}

// LoadConfig searches upwards for a .ralph directory and loads config.yaml from it.
func LoadConfig() (*RalphConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	configDir, err := FindConfigDir(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to find .ralph directory: %w", err)
	}

	config, err := loadConfigFromPath(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		return nil, err
	}

	instructions, commentInstructions, mergeInstructions := loadInstructions(configDir)
	config.Instructions = instructions
	config.CommentInstructions = commentInstructions
	config.MergeInstructions = mergeInstructions

	applyDefaults(config)

	if config.Review.Items != nil || config.Review.Model != "" {
		if err := ValidateReviewConfig(&config.Review); err != nil {
			return nil, fmt.Errorf("invalid review config: %w", err)
		}
	}

	return config, nil
}
