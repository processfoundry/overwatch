package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/christianmscott/overwatch/internal/config"
	"github.com/christianmscott/overwatch/pkg/spec"
)

func cfgPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	return config.DefaultPath
}

func loadCfg() (*spec.Config, error) {
	return config.Load(cfgPath())
}

func serverAddr() (string, error) {
	cfg, err := loadCfg()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%s:%d", cfg.Server.BindAddress, cfg.Server.BindPort), nil
}

var apiClient = &http.Client{Timeout: 5 * time.Second}

func apiDo(method, url string, body any) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return apiClient.Do(req)
}

type apiError struct {
	Error string `json:"error"`
}

func apiReadError(resp *http.Response) error {
	defer resp.Body.Close()
	var ae apiError
	if err := json.NewDecoder(resp.Body).Decode(&ae); err == nil && ae.Error != "" {
		return fmt.Errorf("server: %s", ae.Error)
	}
	return fmt.Errorf("server returned %s", resp.Status)
}
