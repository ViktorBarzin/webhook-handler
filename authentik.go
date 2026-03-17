package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	authentikProvisionPath = "/authentik/provision"

	authentikSecretEnvVar        = "AUTHENTIK_WEBHOOK_SECRET"
	woodpeckerAPIURLEnvVar       = "WOODPECKER_API_URL"
	woodpeckerTokenEnvVar        = "WOODPECKER_TOKEN"
	woodpeckerInfraRepoIDEnvVar  = "WOODPECKER_INFRA_REPO_ID"
)

var (
	authentikWebhookSecret  = os.Getenv(authentikSecretEnvVar)
	woodpeckerAPIURL        = os.Getenv(woodpeckerAPIURLEnvVar)
	woodpeckerToken         = os.Getenv(woodpeckerTokenEnvVar)
	woodpeckerInfraRepoID   = os.Getenv(woodpeckerInfraRepoIDEnvVar)
)

// authentikEvent represents the relevant fields from an Authentik webhook notification.
type authentikEvent struct {
	Body string `json:"body"`
	// Authentik sends event context as a nested structure
	Severity string `json:"severity"`
}

// authentikModelEvent is the parsed body from the notification body text.
// Authentik notification webhooks send a JSON payload with body/severity fields.
// The actual event details (username, email, group) come from the notification context
// which Authentik embeds differently depending on the notification transport version.
// We parse what we need from either query params or the body.

func isAuthentikSignatureValid(r *http.Request, body []byte) bool {
	// Check query param secret first (Authentik notification transports support ?secret=...)
	secret := r.URL.Query().Get("secret")
	if secret != "" && hmac.Equal([]byte(secret), []byte(authentikWebhookSecret)) {
		return true
	}

	// Check X-Authentik-Signature header (HMAC-SHA256 of body)
	sig := r.Header.Get("X-Authentik-Signature")
	if sig == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(authentikWebhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

// triggerWoodpeckerPipeline POSTs to Woodpecker API to trigger the provision-user pipeline.
func triggerWoodpeckerPipeline(username, email string) error {
	if woodpeckerAPIURL == "" || woodpeckerToken == "" || woodpeckerInfraRepoID == "" {
		return fmt.Errorf("woodpecker env vars not configured (need %s, %s, %s)",
			woodpeckerAPIURLEnvVar, woodpeckerTokenEnvVar, woodpeckerInfraRepoIDEnvVar)
	}

	url := fmt.Sprintf("%s/api/repos/%s/pipelines", strings.TrimRight(woodpeckerAPIURL, "/"), woodpeckerInfraRepoID)

	payload := map[string]interface{}{
		"branch": "master",
		"variables": map[string]string{
			"USERNAME":      username,
			"EMAIL":         email,
			"PROVISION_RUN": "true",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal pipeline payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+woodpeckerToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to POST to Woodpecker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("woodpecker API returned %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("Triggered Woodpecker provision pipeline for user=%s email=%s", username, email)
	return nil
}

func authentikProvisionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	log.Printf("Authentik provision webhook received: %s %s (body length: %d)", r.RemoteAddr, r.URL.RequestURI(), len(body))

	// Validate signature/secret
	if authentikWebhookSecret != "" && !isAuthentikSignatureValid(r, body) {
		log.Printf("Authentik webhook: invalid signature from %s", r.RemoteAddr)
		writeError(w, http.StatusForbidden, "invalid signature")
		return
	}

	// Parse the webhook payload
	// Authentik notification webhook payload structure:
	// {
	//   "body": "...",
	//   "severity": "notice",
	//   ...
	// }
	// For custom webhook notifications triggered by expression policies,
	// we expect username and email to be passed as query params or in the body.
	//
	// We support two modes:
	// 1. Query params: ?username=X&email=Y (simplest, set in notification transport URL)
	// 2. JSON body with username/email fields (from custom event context)

	username := r.URL.Query().Get("username")
	email := r.URL.Query().Get("email")

	if username == "" || email == "" {
		// Try parsing from JSON body
		var bodyData map[string]interface{}
		if err := json.Unmarshal(body, &bodyData); err == nil {
			if u, ok := bodyData["username"].(string); ok && username == "" {
				username = u
			}
			if e, ok := bodyData["email"].(string); ok && email == "" {
				email = e
			}

			// Try nested context.model structure (Authentik model_created events)
			if ctx, ok := bodyData["context"].(map[string]interface{}); ok {
				if model, ok := ctx["model"].(map[string]interface{}); ok {
					if app, ok := model["app"].(map[string]interface{}); ok {
						if u, ok := app["username"].(string); ok && username == "" {
							username = u
						}
						if e, ok := app["email"].(string); ok && email == "" {
							email = e
						}
					}
					// Also try flat model fields
					if u, ok := model["username"].(string); ok && username == "" {
						username = u
					}
					if e, ok := model["email"].(string); ok && email == "" {
						email = e
					}
				}
			}
		}
	}

	if username == "" || email == "" {
		log.Printf("Authentik webhook: missing username or email (username=%q, email=%q)", username, email)
		writeError(w, http.StatusBadRequest, "missing username or email — pass as query params or in JSON body")
		return
	}

	// Sanitize: username must be alphanumeric + dashes/underscores
	for _, c := range username {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid character in username: %c", c))
			return
		}
	}

	log.Printf("Authentik provision: triggering pipeline for user=%s email=%s", username, email)

	if err := triggerWoodpeckerPipeline(username, email); err != nil {
		log.Printf("Authentik provision: failed to trigger Woodpecker: %s", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to trigger pipeline: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Provisioning triggered for user %s", username)))
}
