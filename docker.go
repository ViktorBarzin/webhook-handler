package main

import (
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

// Couldn't figure out how to use Dockerhub's webhook verifier
// so instead use a shared url argument as secret
func isDockerHubPayloadValid(r *http.Request, payload docker.BuildPayload) bool {
	if secret, ok := r.URL.Query()[secretArg]; !ok || len(secret) == 0 || secret[0] != webhookSecret {
		return false
	}
	return true
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
		ok := isDockerHubPayloadValid(r, payload.(docker.BuildPayload))
		if !ok {
			writeError(w, 403, fmt.Sprintf("Invalid Dockerhub payload: %s", r.URL.RequestURI()))
			return
		}
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
		err = deploy(namespace[0], deploymentName[0])
		if err != nil {
			writeError(w, 400, err.Error())
		}
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	}
	log.Printf("Request OK: %s %s%s", r.UserAgent(), r.RemoteAddr, r.URL.RequestURI())
}
