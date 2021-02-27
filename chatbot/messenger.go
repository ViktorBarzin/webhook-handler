package chatbot

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const (
	Path = "/fb/webhook"
)

var (
	verifyToken = os.Getenv("FBVerifyToken")
)

func writeError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func ChatbotHandler(w http.ResponseWriter, r *http.Request) {
	// log.Printf("Handling request: %s %s%s\n", r.UserAgent(), r.RemoteAddr, r.URL.RequestURI())

	urlVals := r.URL.Query()
	mode := urlVals.Get("hub.mode")
	token := urlVals.Get("hub.verify_token")
	challenge := urlVals.Get("hub.challenge")
	if mode != "" && token != "" {
		if mode == "subscribe" && token == verifyToken {
			log.Print("webhook verified")
			w.WriteHeader(200)
			w.Write([]byte(challenge))
			return
		} else {
			w.WriteHeader(403)
			return
		}
	}

	bodybytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeError(w, 400, "error reading body")
	}

	body := string(bodybytes)

	log.Printf("%+v\n", body)
}
