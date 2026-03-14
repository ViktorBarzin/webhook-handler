package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"gopkg.in/go-playground/webhooks.v5/docker"
)

const (
	dockerhubPath              = "/dockerhub"
	dockerSecretEnvironmentVar = "WEBHOOKSECRET"

	secretArg         = "s"
	deploymentNameArg = "d"
	namespaceArg      = "n"
)

func derivedSecret(namespace, deployment string) string {
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(namespace + "/" + deployment))
	return hex.EncodeToString(mac.Sum(nil))[:16]
}

func isDockerHubPayloadValid(r *http.Request, namespace, deployment string) bool {
	secret := r.URL.Query().Get(secretArg)
	if secret == "" {
		return false
	}
	expected := derivedSecret(namespace, deployment)
	return hmac.Equal([]byte(secret), []byte(expected))
}

func dockerHubHandler(w http.ResponseWriter, r *http.Request) {
	hook, _ := docker.New()

	log.Printf("Handling request: %s %s%s", r.UserAgent(), r.RemoteAddr, r.URL.RequestURI())
	payload, err := hook.Parse(r, docker.BuildEvent)
	if err != nil {
		log.Printf("Error: " + err.Error())
		return
	}

	switch payload.(type) {
	case docker.BuildPayload:
		namespace, ok := r.URL.Query()[namespaceArg]
		if !ok || len(namespace) == 0 {
			writeError(w, 400, fmt.Sprintf("Argument namespace (%s) not passed or empty", namespaceArg))
			return
		}
		deploymentName, ok := r.URL.Query()[deploymentNameArg]
		if !ok || len(deploymentName) == 0 {
			writeError(w, 400, fmt.Sprintf("Argument deployment name (%s) not passed or empty", deploymentNameArg))
			return
		}
		if len(namespace) > 1 || len(deploymentName) > 1 {
			writeError(w, 400, fmt.Sprintf("Must not specify namespace and deployment name more than once"))
			return
		}

		if !isDockerHubPayloadValid(r, namespace[0], deploymentName[0]) {
			writeError(w, 403, fmt.Sprintf("Invalid Dockerhub payload: %s", r.URL.RequestURI()))
			return
		}
		err = deploy(namespace[0], deploymentName[0])
		if err != nil {
			writeError(w, 400, err.Error())
		}
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	}
	log.Printf("Request OK: %s %s%s", r.UserAgent(), r.RemoteAddr, r.URL.RequestURI())
}
