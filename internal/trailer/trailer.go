// Package trailer formats and parses ralph completion trailers in commit
// messages. Trailer lines are pure strings with no filesystem, git, or network
// access, so they are unit tested directly.
package trailer

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

// base62Alphabet is the alphabet used to encode a completion hash. It is
// ordered digits, then upper-case, then lower-case letters, so encoded hashes
// sort predictably and stay within the alphanumeric run the trailer regex
// expects.
const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// hashLength is the length in characters of an encoded completion hash.
const hashLength = 7

// hashTruncation is the number of SHA-256 digest bytes kept. 40 bits is the
// largest truncation whose every value fits in hashLength base-62 digits:
// 2^40 < 62^7, so encoding the truncated digest never overflows the length.
const hashTruncation = 5

// Hash returns the completion hash for an item's text: a hashLength-character
// base-62 encoding of the SHA-256 digest of the text after normalizing it by
// trimming surrounding whitespace and lower-casing. The same text always
// yields the same hash, and texts differing only in case or surrounding
// whitespace yield the same hash too.
func Hash(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	digest := sha256.Sum256([]byte(normalized))
	var n uint64
	for _, b := range digest[:hashTruncation] {
		n = n<<8 | uint64(b)
	}
	var buf [hashLength]byte
	for i := hashLength - 1; i >= 0; i-- {
		buf[i] = base62Alphabet[n%62]
		n /= 62
	}
	return string(buf[:])
}

// Format renders the completion trailer line for a branch and an item hash:
// "<branch>-<hash>", for example "csv-export-IYAWN02".
func Format(branch string, hash string) string {
	return fmt.Sprintf("%s-%s", branch, hash)
}

// trailerRe matches a whole line of the form "<branch>-<hash>": a non-empty
// branch name and a trailing hashLength-character alphanumeric run joined by a
// hyphen. The branch is any run of git-branch characters, so a branch
// containing hyphens, dots, or slashes still parses; the trailing segment is
// the hash.
var trailerRe = regexp.MustCompile(`(?m)^([A-Za-z0-9][A-Za-z0-9._/-]*)-([A-Za-z0-9]{7})\s*$`)

// Ref is one completion trailer extracted from a commit message: the branch
// and item hash it names.
type Ref struct {
	Branch string
	Hash   string
}

// Parse extracts every completion trailer from a commit message, returning the
// branch and hash each one names, in order of appearance.
func Parse(message string) []Ref {
	matches := trailerRe.FindAllStringSubmatch(message, -1)
	refs := make([]Ref, 0, len(matches))
	for _, m := range matches {
		refs = append(refs, Ref{Branch: m[1], Hash: m[2]})
	}
	return refs
}
