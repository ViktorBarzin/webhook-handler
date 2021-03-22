package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/viktorbarzin/webhook-handler/chatbot"
	"github.com/viktorbarzin/webhook-handler/chatbot/fbapi"

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

	// TEST
	// out, err := executor.Execute(auth.Command{CMD: "echo input isss: $line"}, "kek")
	// if err != nil {
	// 	glog.Fatalf("ERR: %s", err.Error())
	// }
	// glog.Infof(out)
	// return
	// END TEST

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
	chatbotHandler, err := chatbot.NewChatbotHandler(*fsmConfigFile)
	if err != nil {
		glog.Fatalf("Failed to create chatbot handler: %s", err.Error())
	}

	mux := http.NewServeMux()
	mux.HandleFunc(dockerhubPath, dockerHubHandler)
	mux.HandleFunc(fbapi.HandlerPath, chatbotHandler.HandleFunc)

	glog.Infof("Starting webhook handler on %s", listenAddr)
	err = http.ListenAndServe(listenAddr, mux)
	if err != nil {
		glog.Fatalf("Error: %s", err.Error())
	}
}
