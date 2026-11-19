package git

import (
	"fmt"
	"regexp"
)

// credentialPattern matches the password portion of a URL with embedded
// userinfo, such as the installation token in
// url.https://x-access-token:TOKEN@github.com/.insteadOf.
var credentialPattern = regexp.MustCompile(`://[^:/@\s]+:[^@\s/]+@`)

// redactedError wraps err and masks URL credentials in its message, so errors
// that echo command arguments or configuration keys do not leak secrets.
type redactedError struct {
	err error
}

func (e redactedError) Error() string {
	return RedactCredentials(e.err.Error())
}

func (e redactedError) Unwrap() error {
	return e.err
}

// RedactCredentials masks URL credentials in s so error messages and logs do
// not leak secrets such as tokens embedded in a remote URL.
func RedactCredentials(s string) string {
	return credentialPattern.ReplaceAllString(s, "://redacted:redacted@")
}

func Config(global bool, key, value string) error {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key, value)

	_, err := runGit(args...)
	if err != nil {
		// The key and the wrapped git error both echo the arguments, which can
		// carry credentials such as an installation token.
		return fmt.Errorf("failed to set git config %s: %w", RedactCredentials(key), redactedError{err})
	}
	return nil
}

func ConfigList(global bool) (string, error) {
	args := []string{"config", "--list"}
	if global {
		args = append(args, "--global")
	}

	output, err := runGit(args...)
	if err != nil {
		return "", fmt.Errorf("failed to list git config: %w", err)
	}
	return output, nil
}

func ConfigUnset(global bool, key string) error {
	args := []string{"config", "--unset-all"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key)

	_, err := runGit(args...)
	if err != nil {
		return fmt.Errorf("failed to unset git config %s: %w", key, err)
	}
	return nil
}
