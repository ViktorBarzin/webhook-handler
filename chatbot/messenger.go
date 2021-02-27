package chatbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
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

type FbWebhookCallback struct {
	Object string `json:"object"`
	Entry  []struct {
		ID        string `json:"id"`
		Time      int64  `json:"time"`
		Messaging []struct {
			Sender struct {
				ID string `json:"id"`
			} `json:"sender"`
			Recipient struct {
				ID string `json:"id"`
			} `json:"recipient"`
			Timestamp int64 `json:"timestamp"`
			Message   struct {
				Mid  string `json:"mid"`
				Text string `json:"text"`
				Nlp  struct {
					Intents  []interface{} `json:"intents"`
					Entities struct {
						WitLocationLocation []struct {
							ID         string        `json:"id"`
							Name       string        `json:"name"`
							Role       string        `json:"role"`
							Start      int           `json:"start"`
							End        int           `json:"end"`
							Body       string        `json:"body"`
							Confidence float64       `json:"confidence"`
							Entities   []interface{} `json:"entities"`
							Suggested  bool          `json:"suggested"`
							Value      string        `json:"value"`
							Type       string        `json:"type"`
						} `json:"wit$location:location"`
					} `json:"entities"`
					Traits struct {
						WitSentiment []struct {
							ID         string  `json:"id"`
							Value      string  `json:"value"`
							Confidence float64 `json:"confidence"`
						} `json:"wit$sentiment"`
						WitGreetings []struct {
							ID         string  `json:"id"`
							Value      string  `json:"value"`
							Confidence float64 `json:"confidence"`
						} `json:"wit$greetings"`
					} `json:"traits"`
					DetectedLocales []struct {
						Locale     string  `json:"locale"`
						Confidence float64 `json:"confidence"`
					} `json:"detected_locales"`
				} `json:"nlp"`
			} `json:"message"`
		} `json:"messaging"`
	} `json:"entry"`
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
	requestDump, err := httputil.DumpRequest(r, true)
	log.Printf("Processing: '%+v'", string(requestDump))
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
		}
		w.WriteHeader(403)
		return
	}

	bodybytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeError(w, 400, "error reading body")
	}

	var response FbWebhookCallback
	json.Unmarshal(bodybytes, &response)

	for _, e := range response.Entry {
		for _, m := range e.Messaging {
			log.Printf("Message: %s", m.Message.Text)
			sendMessage(fmt.Sprintf("You sent me: %s", m.Message.Text), m.Sender.ID)
		}
	}
	// log.Printf("%+v\n", response.)
}

func Main() {
	uid := "3804650372987546"
	sendMessage("test kek", uid)
}
