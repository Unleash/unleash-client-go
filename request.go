package unleash

import (
	"encoding/json"
	"github.com/unleash/unleash-client-go/internal/api"
	"net/http"
	"net/url"
)

func get(client *http.Client, url url.URL, etag, appName, instanceId string) error {
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Add("UNLEASH-APPNAME", appName)
	req.Header.Add("UNLEASH-INSTANCEID", instanceId)

	if etag != "" {
		req.Header.Add("If-None-Match", etag)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	var featureResp api.FeatureResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&featureResp); err != nil {
		return err
	}

	return nil
}
