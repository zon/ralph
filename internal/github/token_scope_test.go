package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenWorkflowScope(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		status      int
		scopes      []string
		wantPresent bool
		wantKnown   bool
		wantErr     bool
	}{
		{
			name:        "classic token with workflow scope",
			token:       "gho_test",
			status:      http.StatusOK,
			scopes:      []string{"repo, workflow, gist"},
			wantPresent: true,
			wantKnown:   true,
		},
		{
			name:        "classic token without workflow scope",
			token:       "gho_test",
			status:      http.StatusOK,
			scopes:      []string{"repo, gist"},
			wantPresent: false,
			wantKnown:   true,
		},
		{
			name:        "scopes spread across headers",
			token:       "gho_test",
			status:      http.StatusOK,
			scopes:      []string{"repo", "workflow"},
			wantPresent: true,
			wantKnown:   true,
		},
		{
			name:        "token type without reported scopes",
			token:       "github_pat_test",
			status:      http.StatusOK,
			scopes:      nil,
			wantPresent: false,
			wantKnown:   false,
		},
		{
			name:        "empty token",
			token:       "",
			status:      http.StatusOK,
			wantPresent: false,
			wantKnown:   false,
		},
		{
			name:    "rejected token",
			token:   "gho_test",
			status:  http.StatusUnauthorized,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/user", r.URL.Path)
				for _, scope := range tt.scopes {
					w.Header().Add("X-OAuth-Scopes", scope)
				}
				w.WriteHeader(tt.status)
			}))
			t.Cleanup(server.Close)

			oldClient := httpClient
			httpClient = &http.Client{
				Transport: &rewriteTransport{
					old:       http.DefaultTransport,
					serverURL: strings.TrimPrefix(server.URL, "http://"),
				},
			}
			t.Cleanup(func() { httpClient = oldClient })

			present, known, err := TokenWorkflowScope(context.Background(), tt.token)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPresent, present)
			assert.Equal(t, tt.wantKnown, known)
		})
	}
}
