package config

import (
	_ "embed"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

//go:embed fix-service-instructions.md
var defaultFixServiceInstructions string

//go:embed config.md
var configDocumentation string

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

// SecretKeyRef references a key within a named Kubernetes Secret.
type SecretKeyRef struct {
	Name string `yaml:"name"`
	Key  string `yaml:"key"`
}

// EnvVar is a single entry in the workflow env section: either a literal
// string value or a reference to a key in a Kubernetes Secret.
type EnvVar struct {
	Value        string
	SecretKeyRef *SecretKeyRef
}

// UnmarshalYAML implements yaml.Unmarshaler so each env entry accepts either a
// literal string value or a mapping holding a secretKeyRef.
func (e *EnvVar) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var value string
		if err := node.Decode(&value); err != nil {
			return err
		}
		e.Value = value
		e.SecretKeyRef = nil
		return nil
	case yaml.MappingNode:
		if len(node.Content) != 2 || node.Content[0].Value != "secretKeyRef" {
			return fmt.Errorf("line %d: env entry must be a string or a mapping with a secretKeyRef", node.Line)
		}
		if node.Content[1].Kind != yaml.MappingNode {
			return fmt.Errorf("line %d: env entry secretKeyRef must be a mapping with a name and key", node.Line)
		}
		var ref SecretKeyRef
		if err := node.Content[1].Decode(&ref); err != nil {
			return err
		}
		e.Value = ""
		e.SecretKeyRef = &ref
		return nil
	}
	return fmt.Errorf("line %d: env entry must be a string or a mapping with a secretKeyRef", node.Line)
}

// MarshalYAML implements yaml.Marshaler so literal values round-trip as bare
// strings and secretKeyRef entries as mappings.
func (e EnvVar) MarshalYAML() (interface{}, error) {
	if e.SecretKeyRef != nil {
		return map[string]interface{}{"secretKeyRef": e.SecretKeyRef}, nil
	}
	return e.Value, nil
}

// ResourceList holds the memory and CPU quantities of a single resources entry.
type ResourceList struct {
	Memory string `yaml:"memory,omitempty"`
	CPU    string `yaml:"cpu,omitempty"`
}

// WorkflowResources holds the CPU and memory requests and limits for the ralph
// executor container.
type WorkflowResources struct {
	Requests ResourceList `yaml:"requests,omitempty"`
	Limits   ResourceList `yaml:"limits,omitempty"`
}

// ValidateWorkflowResources returns an error when any configured resource
// quantity is malformed or a limit is set below its request. An empty
// resources block is valid.
func ValidateWorkflowResources(r WorkflowResources) error {
	for _, list := range []struct {
		label string
		value ResourceList
	}{
		{"request", r.Requests},
		{"limit", r.Limits},
	} {
		for _, quantity := range []struct {
			resource string
			value    string
		}{
			{"memory", list.value.Memory},
			{"cpu", list.value.CPU},
		} {
			if quantity.value == "" {
				continue
			}
			if _, err := parseResourceQuantity(quantity.value); err != nil {
				return fmt.Errorf("invalid %s %s %q", quantity.resource, list.label, quantity.value)
			}
		}
	}

	for _, resource := range []struct {
		name           string
		request, limit string
	}{
		{"memory", r.Requests.Memory, r.Limits.Memory},
		{"cpu", r.Requests.CPU, r.Limits.CPU},
	} {
		if resource.request == "" || resource.limit == "" {
			continue
		}
		request, err := parseResourceQuantity(resource.request)
		if err != nil {
			return err
		}
		limit, err := parseResourceQuantity(resource.limit)
		if err != nil {
			return err
		}
		if limit < request {
			return fmt.Errorf("%s limit %q is below its request %q", resource.name, resource.limit, resource.request)
		}
	}

	return nil
}

// resourceQuantitySuffixes maps resource quantity suffixes to their multiplier
// in the base unit. Binary suffixes (Ki, Mi, ...) multiply by powers of 1024;
// SI suffixes multiply by powers of 1000.
var resourceQuantitySuffixes = map[string]float64{
	"n":  1e-9,
	"u":  1e-6,
	"m":  1e-3,
	"k":  1e3,
	"K":  1e3,
	"M":  1e6,
	"G":  1e9,
	"T":  1e12,
	"P":  1e15,
	"E":  1e18,
	"Ki": 1024,
	"Mi": 1024 * 1024,
	"Gi": 1024 * 1024 * 1024,
	"Ti": 1024 * 1024 * 1024 * 1024,
	"Pi": 1024 * 1024 * 1024 * 1024 * 1024,
	"Ei": 1024 * 1024 * 1024 * 1024 * 1024 * 1024,
}

// resourceQuantityRE matches a Kubernetes resource quantity: an optional sign,
// a number, an optional decimal exponent, and an optional suffix. The groups
// capture the number, exponent, and suffix separately.
var resourceQuantityRE = regexp.MustCompile(`^([+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+))((?:[eE][+-]?[0-9]+)?)((?:Ei|Ki|Mi|Gi|Ti|Pi|n|u|m|k|K|M|G|T|P|E)?)$`)

// parseResourceQuantity parses a Kubernetes resource quantity and returns its
// value in the base unit. It rejects unknown suffixes, malformed numbers, and
// quantities that combine an exponent with a suffix.
func parseResourceQuantity(q string) (float64, error) {
	match := resourceQuantityRE.FindStringSubmatch(q)
	if match == nil {
		return 0, fmt.Errorf("invalid quantity %q", q)
	}

	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid quantity %q", q)
	}

	if exponent := match[2]; exponent != "" {
		if match[3] != "" {
			return 0, fmt.Errorf("invalid quantity %q", q)
		}
		exp, err := strconv.Atoi(exponent[1:])
		if err != nil {
			return 0, fmt.Errorf("invalid quantity %q", q)
		}
		return value * math.Pow10(exp), nil
	}

	if match[3] == "" {
		return value, nil
	}

	multiplier, ok := resourceQuantitySuffixes[match[3]]
	if !ok {
		return 0, fmt.Errorf("invalid quantity %q", q)
	}
	return value * multiplier, nil
}

// WorkflowConfig represents Argo Workflow configuration options
type WorkflowConfig struct {
	Image      ImageConfig       `yaml:"image,omitempty"`
	ConfigMaps []ConfigMapMount  `yaml:"configMaps,omitempty"`
	Secrets    []SecretMount     `yaml:"secrets,omitempty"`
	Env        map[string]EnvVar `yaml:"env,omitempty"`
	Context    string            `yaml:"context,omitempty"`
	Namespace  string            `yaml:"namespace,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty"`
	Resources  WorkflowResources `yaml:"resources,omitempty"`
}

// UnmarshalYAML implements yaml.Unmarshaler so a null workflow.env entry is
// rejected with a descriptive error. yaml.v3 decodes a null env entry as an
// empty EnvVar without calling EnvVar.UnmarshalYAML, so the null check runs
// here over the workflow mapping node before normal decoding.
func (w *WorkflowConfig) UnmarshalYAML(node *yaml.Node) error {
	if err := rejectNullEnvEntries(node); err != nil {
		return err
	}
	type plain WorkflowConfig
	var p plain
	if err := node.Decode(&p); err != nil {
		return err
	}
	*w = WorkflowConfig(p)
	return nil
}

// rejectNullEnvEntries returns an error when a workflow.env entry has no value
// (a YAML null), which yaml.v3 would otherwise silently decode as an empty
// EnvVar. YAML aliases are unwrapped so anchored nulls are rejected too.
func rejectNullEnvEntries(node *yaml.Node) error {
	envNode := mappingValue(node, "env")
	if envNode == nil {
		return nil
	}
	if envNode.Kind == yaml.AliasNode {
		envNode = envNode.Alias
	}
	if envNode.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(envNode.Content); i += 2 {
		valueNode := envNode.Content[i+1]
		if valueNode.Kind == yaml.AliasNode {
			valueNode = valueNode.Alias
		}
		if valueNode.Kind == yaml.ScalarNode && valueNode.Tag == "!!null" {
			return fmt.Errorf("line %d: env entry must be a string or a mapping with a secretKeyRef", valueNode.Line)
		}
	}
	return nil
}

// mappingValue returns the value node for key in a mapping node, or nil.
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

const LoopTypeDomainFunction = "domain-function"

var validLoopTypes = map[string]bool{
	LoopTypeDomainFunction: true,
}

const (
	ModeLocal    = "local"
	ModeWorktree = "worktree"
	ModeRemote   = "remote"
	DefaultMode  = ModeLocal
)

var validModes = map[string]bool{
	ModeLocal:    true,
	ModeWorktree: true,
	ModeRemote:   true,
}

// ValidateMode returns an error when mode is not one of local, worktree, or
// remote.
func ValidateMode(mode string) error {
	if !validModes[mode] {
		return fmt.Errorf("invalid mode: %s (expected local, worktree, or remote)", mode)
	}
	return nil
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

// ValidateConfig represents the validate-specific configuration
type ValidateConfig struct {
	Model string `yaml:"model,omitempty"`
}

// LoopConfig represents a named loop configuration with a slug, steps, and an
// optional iteration cap.
type LoopConfig struct {
	Slug  string   `yaml:"slug"`
	Steps []string `yaml:"steps"`
	Max   *int     `yaml:"max,omitempty"`
}

// RalphConfig represents the .ralph/config.yaml structure
type RalphConfig struct {
	Variant         string         `yaml:"variant,omitempty"`
	Mode            string         `yaml:"mode,omitempty"`    // Execution mode: local, worktree, or remote (default: local)
	Items           string         `yaml:"items,omitempty"`   // jq query selecting the item array from a project file (default: .)
	Cleanup         bool           `yaml:"cleanup,omitempty"` // Delete the project file once every item is complete (default: false)
	Base            string         `yaml:"-"`                 // Base branch resolved by the caller, bounding the commit log completion is read from
	ExtraIterations *int           `yaml:"extraIterations,omitempty"`
	DefaultBranch   string         `yaml:"defaultBranch,omitempty"`
	Model           string         `yaml:"model,omitempty"` // AI model to use for coding and PR summary (default: deepseek/deepseek-chat)
	Agent           string         `yaml:"agent,omitempty"` // opencode agent to use for coding (default: opencode's primary agent)
	Before          []Before       `yaml:"before,omitempty"`
	Services        []Service      `yaml:"services,omitempty"`
	Workflow        WorkflowConfig `yaml:"workflow,omitempty"`
	App             AppInfo        `yaml:"app,omitempty"`
	Review          ReviewConfig   `yaml:"review,omitempty"`
	Validate        ValidateConfig `yaml:"validate,omitempty"`
	Loops           []LoopConfig   `yaml:"loops,omitempty"`
	ConfigPath      string         `yaml:"-"` // Path to the loaded config file
	Instructions    string         `yaml:"-"` // Not persisted in YAML, loaded from .ralph/instructions.md
}

func DefaultFixServiceInstructions() string {
	return defaultFixServiceInstructions
}

// ConfigDocumentation returns the embedded configuration reference shown by
// `ralph help config`.
func ConfigDocumentation() string {
	return configDocumentation
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

// validateLoopConfigs rejects loop entries whose max is set but not a positive
// integer, naming the offending loop's slug.
func validateLoopConfigs(loops []LoopConfig) error {
	for _, loop := range loops {
		if loop.Max != nil && *loop.Max <= 0 {
			return fmt.Errorf("loop %q max must be a positive integer", loop.Slug)
		}
	}
	return nil
}

// validateWorkflowEnv rejects env entries whose secretKeyRef lacks a name or key.
func validateWorkflowEnv(env map[string]EnvVar) error {
	for name, envVar := range env {
		if envVar.SecretKeyRef == nil {
			continue
		}
		if envVar.SecretKeyRef.Name == "" {
			return fmt.Errorf("env %q: secretKeyRef requires a name", name)
		}
		if envVar.SecretKeyRef.Key == "" {
			return fmt.Errorf("env %q: secretKeyRef requires a key", name)
		}
	}
	return nil
}

// applyDefaults fills in zero-value fields with their default values.
func applyDefaults(config *RalphConfig) {
	if config.Items == "" {
		config.Items = "."
	}
	if config.Mode == "" {
		config.Mode = DefaultMode
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

// ResolveItems returns the effective item query for a run: the flag when
// passed, otherwise the config `items` field, otherwise the default ".".
func (c *RalphConfig) ResolveItems(flag string) string {
	if flag != "" {
		return flag
	}
	if c.Items != "" {
		return c.Items
	}
	return "."
}

// ResolveCleanup returns whether project file cleanup is enabled for a run:
// the flag when passed, otherwise the config `cleanup` field, otherwise false.
// A nil flag means the flag was not passed.
func (c *RalphConfig) ResolveCleanup(flag *bool) bool {
	if flag != nil {
		return *flag
	}
	return c.Cleanup
}

// ResolveMode returns the effective execution mode for a run: the flag when
// passed, otherwise the config `mode` field, otherwise local. A mode value
// other than local, worktree, or remote is rejected with a descriptive error.
func (c *RalphConfig) ResolveMode(flag string) (string, error) {
	mode := c.Mode
	if flag != "" {
		mode = flag
	}
	if mode == "" {
		mode = DefaultMode
	}
	if err := ValidateMode(mode); err != nil {
		return "", err
	}
	return mode, nil
}

// LoopSteps returns the steps of the first loop config matching the slug, or
// an error when none matches.
func (c *RalphConfig) LoopSteps(slug string) ([]string, error) {
	for _, loop := range c.Loops {
		if loop.Slug == slug {
			return loop.Steps, nil
		}
	}
	return nil, fmt.Errorf("loop config not found: %s", slug)
}

// LoopMax returns the configured iteration cap of the first loop config
// matching the slug, or nil when no entry matches or no entry sets a max.
func (c *RalphConfig) LoopMax(slug string) *int {
	for _, loop := range c.Loops {
		if loop.Slug == slug {
			return loop.Max
		}
	}
	return nil
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

// loadInstructions loads the instruction file from the config directory.
// Development instructions are left empty when .ralph/instructions.md is
// absent, so the prompt supplies its own default steps.
func loadInstructions(configDir string) string {
	instructionsPath := filepath.Join(configDir, "instructions.md")
	var instructions string
	if instructionsData, err := os.ReadFile(instructionsPath); err == nil {
		instructions = string(instructionsData)
	}
	return instructions
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

	config.Instructions = loadInstructions(configDir)

	applyDefaults(config)

	if err := ValidateMode(config.Mode); err != nil {
		return nil, err
	}

	if config.Review.Items != nil || config.Review.Model != "" {
		if err := ValidateReviewConfig(&config.Review); err != nil {
			return nil, fmt.Errorf("invalid review config: %w", err)
		}
	}

	if err := validateWorkflowEnv(config.Workflow.Env); err != nil {
		return nil, fmt.Errorf("invalid workflow env: %w", err)
	}

	if err := validateLoopConfigs(config.Loops); err != nil {
		return nil, err
	}

	return config, nil
}
