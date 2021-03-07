package main

import (
	"flag"
	"net/http"
	"os"

	"viktorbarzin/webhook-handler/chatbot"

	"github.com/golang/glog"
)

const (
	fsmFlagName      = "fsm"
	listenAddr       = ":3000"
	configEnvVarName = "CONFIG"
)

func main() {
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING")
	flag.Set("v", "2")
	fsmConfigFile := flag.String(fsmFlagName, "", "YAML file which contains the description of conversation state machine.")
	flag.Parse()

	var configFromEnv string
	if *fsmConfigFile == "" {
		configFromEnv = os.Getenv(configEnvVarName)
		if configFromEnv == "" {
			glog.Fatal("Please provide config file(--" + fsmFlagName + " flag) or set " + configEnvVarName + " env variable")
		} else {
			*fsmConfigFile = configFromEnv
		}
	}

	glog.Infof("Initializing chatbot handler with %s config file", *fsmConfigFile)
	chatbotHandler := chatbot.NewChatbotHandler(*fsmConfigFile)

	mux := http.NewServeMux()
	mux.HandleFunc(dockerhubPath, dockerHubHandler)
	mux.HandleFunc(chatbot.Path, chatbotHandler.HandleFunc)

	glog.Infof("Starting webhook handler on %s", listenAddr)
	http.ListenAndServe(listenAddr, mux)

	// Testing
	// chatbot.Main()
	// statemachine.Main()
}
