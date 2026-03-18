package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	authentikProvisionPath = "/authentik/provision"

	authentikSecretEnvVar       = "AUTHENTIK_WEBHOOK_SECRET"
	woodpeckerAPIURLEnvVar      = "WOODPECKER_API_URL"
	woodpeckerTokenEnvVar       = "WOODPECKER_TOKEN"
	woodpeckerInfraRepoIDEnvVar = "WOODPECKER_INFRA_REPO_ID"

	maxBodySize     = 1 << 20 // 1 MB
	maxUsernameLen  = 63      // Kubernetes name limit
)

var (
	authentikWebhookSecret = os.Getenv(authentikSecretEnvVar)
	woodpeckerAPIURL       = os.Getenv(woodpeckerAPIURLEnvVar)
	woodpeckerToken        = os.Getenv(woodpeckerTokenEnvVar)
	woodpeckerInfraRepoID  = os.Getenv(woodpeckerInfraRepoIDEnvVar)

	woodpeckerClient = &http.Client{Timeout: 30 * time.Second}
)

func isAuthentikSignatureValid(r *http.Request, body []byte) bool {
	secret := r.URL.Query().Get("secret")
	if secret != "" && hmac.Equal([]byte(secret), []byte(authentikWebhookSecret)) {
		return true
	}

	sig := r.Header.Get("X-Authentik-Signature")
	if sig == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(authentikWebhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

func triggerWoodpeckerPipeline(username, email string) error {
	if woodpeckerAPIURL == "" || woodpeckerToken == "" || woodpeckerInfraRepoID == "" {
		return fmt.Errorf("woodpecker env vars not configured")
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

	resp, err := woodpeckerClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to POST to Woodpecker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
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

	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	log.Printf("Authentik provision webhook received: %s %s (body length: %d)", r.RemoteAddr, r.URL.RequestURI(), len(body))

	// Reject requests when no secret is configured (fail-closed)
	if authentikWebhookSecret == "" {
		log.Printf("Authentik webhook: secret not configured, rejecting request")
		writeError(w, http.StatusInternalServerError, "webhook secret not configured")
		return
	}

	if !isAuthentikSignatureValid(r, body) {
		log.Printf("Authentik webhook: invalid signature from %s", r.RemoteAddr)
		writeError(w, http.StatusForbidden, "invalid signature")
		return
	}

	username := r.URL.Query().Get("username")
	email := r.URL.Query().Get("email")

	if username == "" || email == "" {
		var bodyData map[string]interface{}
		if err := json.Unmarshal(body, &bodyData); err == nil {
			if u, ok := bodyData["username"].(string); ok && username == "" {
				username = u
			}
			if e, ok := bodyData["email"].(string); ok && email == "" {
				email = e
			}

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

	// Validate username: alphanumeric + dashes/underscores, max length
	if len(username) > maxUsernameLen {
		writeError(w, http.StatusBadRequest, "username too long")
		return
	}
	for _, c := range username {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			writeError(w, http.StatusBadRequest, "invalid character in username")
			return
		}
	}

	log.Printf("Authentik provision: triggering pipeline for user=%s email=%s", username, email)

	if err := triggerWoodpeckerPipeline(username, email); err != nil {
		log.Printf("Authentik provision: failed to trigger Woodpecker: %s", err)
		writeError(w, http.StatusInternalServerError, "failed to trigger provisioning pipeline")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Provisioning triggered for user %s", username)))
}
