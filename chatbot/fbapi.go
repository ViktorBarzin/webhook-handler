package chatbot

import (
	"bytes"
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
	fbAPIURI = "https://graph.facebook.com/v2.6/me/messages"
	Path     = "/fb/webhook"
)

var (
	verifyToken = os.Getenv("FB_VERIFY_TOKEN")
	pageToken   = os.Getenv("FB_PAGE_TOKEN")
	appSecret   = os.Getenv("FB_APP_SECRET")
	testEnv     = os.Getenv("TEST")
)

func writeError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func sendRawMessage(receiverPsid, msg string) (*http.Response, error) {
	payload := getRawMessagePayload(receiverPsid, msg)
	reader, err := payloadReader(payload, receiverPsid)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get raw message payload reader for msg: %s", msg)
	}
	resp, err := sendRequest(reader)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send request with msg '%s' to uid '%s'", msg, receiverPsid)
	}
	return resp, nil
}

func sendPostBackMessage(receiverPsid string, payload models.PayloadPostback) (*http.Response, error) {
	reader, err := payloadReader(payload, receiverPsid)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to create message reader for message: %+v", payload))
	}
	resp, err := sendRequest(reader)
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

func payloadReader(msg interface{}, receiverPsid string) (io.Reader, error) {
	payloadBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request data")
	}
	glog.Infof("Sending: %s", string(payloadBytes))
	body := bytes.NewReader(payloadBytes)
	return body, nil
}

func sendRequest(body io.Reader) (*http.Response, error) {
	return sendRequestURI(fbAPIURI, body)
}

func sendRequestURI(uri string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", uri+"?access_token="+pageToken, body)
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
