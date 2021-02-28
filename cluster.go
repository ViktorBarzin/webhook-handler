package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	kubernetesBaseURI = "https://kubernetes:6443"
	tokenPath         = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

var (
	webhookSecret string = os.Getenv(dockerSecretEnvironmentVar)
)

// try: cat /var/run/secrets/kubernetes.io/serviceaccount/token
func authToken() string {
	f, err := ioutil.ReadFile(tokenPath)
	if err != nil {
		panic(fmt.Sprintf("cannot read token file %s", tokenPath))
	}
	return string(f)
}

// Send PATCH request to the k8s API to update deployment
func deploy(namespace, deployment string) error {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	params := patchArgs()
	req, err := http.NewRequest("PATCH", kubernetesDeploymentEndpoint(namespace, deployment), bytes.NewBuffer([]byte(params)))
	if err != nil {
		return fmt.Errorf("error building request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken()))
	req.Header.Set("Accept", "application/json, */*")
	req.Header.Set("Content-Type", "application/strategic-merge-patch+json")

	log.Printf("DEBUG:sending request: %v", req)
	resp, err := client.Do(req)
	log.Printf("DEBUG:request response: %v", resp)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body := new(strings.Builder)
		_, _ = io.Copy(body, resp.Body)
		return fmt.Errorf("error interacting with k8s API: %s", body.String())
	}
	return nil
}

func patchArgs() string {
	now := fmt.Sprint(time.Now().Format(time.RFC3339))
	res := fmt.Sprintf(`{
    "spec": {
        "template": {
            "metadata": {
                "annotations": {
                    "kubectl.kubernetes.io/restartedAt": "%s"
                }
            }
        }
    }
}`, now)
	return res
}

func kubernetesDeploymentEndpoint(namespace, deployment string) string {
	return fmt.Sprintf("%s/apis/apps/v1/namespaces/%s/deployments/%s", kubernetesBaseURI, namespace, deployment)
}
