package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrNoCredentials reports that OpenCode holds no stored credentials.
var ErrNoCredentials = errors.New("OpenCode holds no credentials, authenticate with 'opencode auth login'")

// ExportCredentials returns the credentials OpenCode stores as the JSON
// document `opencode auth export` prints. OpenCode 2 keeps credentials in its
// database rather than the auth.json file OpenCode 1 used, so this is how ralph
// reads them. It returns ErrNoCredentials when OpenCode stores none.
func ExportCredentials() ([]byte, error) {
	var stdout, stderr bytes.Buffer
	if err := execOpenCode(context.Background(), []string{"auth", "export", "--standalone"}, &stdout, &stderr); err != nil {
		return nil, withStderr(err, &stderr)
	}
	return parseExportedCredentials(stdout.Bytes())
}

// ImportCredentials loads a document produced by ExportCredentials into
// OpenCode's store.
func ImportCredentials(path string) error {
	return execOpenCode(context.Background(), []string{"auth", "import", "--standalone", path}, os.Stdout, os.Stderr)
}

// parseExportedCredentials validates that data is the credential array
// `opencode auth export` emits and returns it without surrounding whitespace.
func parseExportedCredentials(data []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(data)
	var credentials []json.RawMessage
	if err := json.Unmarshal(trimmed, &credentials); err != nil {
		return nil, fmt.Errorf("failed to parse OpenCode credentials: %w", err)
	}
	if len(credentials) == 0 {
		return nil, ErrNoCredentials
	}
	return trimmed, nil
}

// withStderr appends the captured stderr to err so a failed command reports why.
func withStderr(err error, stderr *bytes.Buffer) error {
	if message := strings.TrimSpace(stderr.String()); message != "" {
		return fmt.Errorf("%w: %s", err, message)
	}
	return err
}
