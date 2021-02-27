package chatbot

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

const (
	fbAPIURI = "https://graph.facebook.com/v2.6/me/messages"
	Path     = "/fb/webhook"
)

var (
	verifyToken = os.Getenv("FB_VERIFY_TOKEN")
	pageToken   = os.Getenv("FB_PAGE_TOKEN")
)

type Payload struct {
	Recipient Recipient `json:"recipient"`
	Message   Message   `json:"message"`
}
type Recipient struct {
	ID string `json:"id"`
}
type Message struct {
	Text string `json:"text"`
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func sendMessage(msg, receiverPsid string) (*http.Response, error) {
	data := Payload{
		Recipient: Recipient{ID: receiverPsid},
		Message:   Message{Text: msg},
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request data")
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", fbAPIURI+"?access_token="+pageToken, body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create POST request struct")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send POST request to "+fbAPIURI)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	log.Printf("Response body: %+v", string(respBody))
	return resp, nil
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

func Main() {
	uid := "3804650372987546"
	sendMessage("test kek", uid)
}
