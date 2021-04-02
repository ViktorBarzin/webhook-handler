package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/viktorbarzin/webhook-handler/chatbot/fbapi"
)

// Direct message Viktor
// This is fo emergencies only and may not work

const (
	messageViktorHandler = "/fb/message-viktor"
	viktorPSID           = "3804650372987546"
)

func MessageViktorHandleFunc(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeError(w, 400, "failed to read body")
		return
	}
	if err := fbapi.SendRawMessage(viktorPSID, string(body)); err != nil {
		writeError(w, 400, fmt.Sprintf("failed to send '%s' to viktor: %s", string(body), err.Error()))
	}
}
