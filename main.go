// webhook-handler listens for webhooks from GitHub (for now)
// and upon `push` and pull request merge events go master redeploys a given k8s resource
package main

import (
	"log"
	"net/http"
	"os"

	"viktorbarzin/webhook-handler/chatbot"
)

const (
	listenAddr = ":3000"
)

func main() {
	webhookSecret = os.Getenv(dockerSecretEnvironmentVar)
	if len(webhookSecret) == 0 {
		log.Printf("WARNING: webhook secret environment variable is empty. Anyone can redeploy ANY deployment!")
	}
	http.HandleFunc(dockerhubPath, dockerHubHandler)
	http.HandleFunc(chatbot.Path, chatbot.ChatbotHandler)

	log.Printf("Starting webhook handler on %s", listenAddr)
	http.ListenAndServe(listenAddr, nil)
}
