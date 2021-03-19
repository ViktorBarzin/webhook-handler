package fbapi

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"viktorbarzin/webhook-handler/chatbot/models"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

const (
	fbAPIURI    = "https://graph.facebook.com/v2.6/me/messages"
	HandlerPath = "/fb/webhook"

	GetStartedMessage = "GetStarted"
)

var (
	VerifyToken = os.Getenv("FB_VERIFY_TOKEN")
	pageToken   = os.Getenv("FB_PAGE_TOKEN")
	AppSecret   = os.Getenv("FB_APP_SECRET")
	TestEnv     = os.Getenv("TEST")
)

func ResponseWrite(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func SendRawMessage(receiverPsid, msg string) (*http.Response, error) {
	payload := getRawMessagePayload(receiverPsid, msg)
	reader, err := payloadReader(payload, receiverPsid)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get raw message payload reader for msg: %s", msg)
	}
	resp, err := SendRequest(reader)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send request with msg '%s' to uid '%s'", msg, receiverPsid)
	}
	return resp, nil
}

func SendPostBackMessage(receiverPsid string, payload models.PayloadPostback) (*http.Response, error) {
	reader, err := payloadReader(payload, receiverPsid)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to create message reader for message: %+v", payload))
	}
	resp, err := SendRequest(reader)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send payload: '%+v' to user: %s", payload, receiverPsid)
	}
	return resp, nil
}

func getRawMessagePayload(receiver, msg string) models.Payload {
	return models.Payload{
		Recipient: models.Recipient{ID: receiver},
		Message: models.Message{
			Text: msg,
		},
	}
}

func payloadReader(msg interface{}, receiverPsid string) (string, error) {
	payloadBytes, err := json.Marshal(msg)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal request data")
	}
	// glog.Infof("Sending: %s", string(payloadBytes))
	return string(payloadBytes), nil
}

func SendRequest(body string) (*http.Response, error) {
	return SendRequestURI(fbAPIURI, body)
}

func SendRequestURI(uri string, body string) (*http.Response, error) {
	req, err := http.NewRequest("POST", uri+"?access_token="+pageToken, bytes.NewReader([]byte(body)))
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

func ValidSignature(r *http.Request) (bool, string) {
	var buf bytes.Buffer
	cloned := io.TeeReader(r.Body, &buf)
	// if in testing, return true
	if TestEnv != "" {
		return true, "test mode"
	}
	signatureValues, ok := r.Header["X-Hub-Signature"]
	if !ok {
		return false, "'X-Hub-Signature' header is not set"
	}
	if len(signatureValues) == 0 || len(signatureValues) > 1 {
		return false, fmt.Sprintf("'X-Hub-Signature' must have exactly 1 value. got %d values", len(signatureValues))
	}
	signature := signatureValues[0]
	if len(signature) < 5 || signature[0:5] != "sha1=" {
		return false, fmt.Sprintf("invalid format of signature. expected: 'sha1=SIGNATURE_VALUE', received %s", signature)
	}
	signature = signature[5:]

	postData, err := ioutil.ReadAll(cloned)
	r.Body = ioutil.NopCloser(&buf)
	if err != nil {
		return false, "failed to get body for which to calculate hmac"
	}
	h := hmac.New(sha1.New, []byte(AppSecret))
	h.Write([]byte(postData))

	expected := hex.EncodeToString(h.Sum(nil))
	matching := expected == signature
	if !matching {
		return false, fmt.Sprintf("signature are not matching. got signature %s", signature)
	}
	return true, "signatures are matching"
}

func IsVerifyRequest(w http.ResponseWriter, r *http.Request) bool {
	urlVals := r.URL.Query()
	mode := urlVals.Get("hub.mode")
	token := urlVals.Get("hub.verify_token")
	challenge := urlVals.Get("hub.challenge")
	if mode != "" && token != "" {
		if mode == "subscribe" && token == VerifyToken {
			glog.Info("webhook verified")
			w.WriteHeader(200)
			w.Write([]byte(challenge))
		} else {
			w.WriteHeader(403)
		}
		return true
	}
	return false
}

func SetGetStartedButton() error {
	getStartedButtonPayload := map[string]map[string]string{
		"get_started": {"payload": GetStartedMessage},
	}
	marshalled, err := json.Marshal(getStartedButtonPayload)
	if err != nil {
		return errors.Wrap(err, "failed to marshall get started button payload")
	}
	resp, err := SendRequestURI("https://graph.facebook.com/v2.6/me/messenger_profile", string(marshalled))
	if err != nil {
		return errors.Wrap(err, "failed sending request")
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed reading response body")
	}
	glog.Infof("Received response to setting payload button: '%s'", respBody)
	return nil
}
