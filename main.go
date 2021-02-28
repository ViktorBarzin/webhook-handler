package main

import (
	"flag"
	"net/http"

	"viktorbarzin/webhook-handler/chatbot"

	"github.com/golang/glog"
)

const (
	listenAddr = ":3000"
)

func main() {
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING")
	flag.Set("v", "2")
	flag.Parse()

	chatbotHandler := chatbot.NewChatbotHandler()

	mux := http.NewServeMux()
	mux.HandleFunc(dockerhubPath, dockerHubHandler)
	mux.HandleFunc(chatbot.Path, chatbotHandler.HandleFunc)

	glog.Infof("Starting webhook handler on %s", listenAddr)
	http.ListenAndServe(listenAddr, mux)

	// Testing
	// chatbot.Main()
	// statemachine.Main()
}
