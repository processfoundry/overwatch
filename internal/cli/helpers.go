package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/christianmscott/overwatch/internal/auth"
	"github.com/christianmscott/overwatch/internal/config"
	"github.com/christianmscott/overwatch/pkg/spec"
	"gopkg.in/yaml.v3"
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

func loadClientConfig() (*spec.ClientConfig, error) {
	dir := clientDir()
	data, err := os.ReadFile(filepath.Join(dir, "client.yaml"))
	if err != nil {
		return nil, err
	}
	var cc spec.ClientConfig
	if err := yaml.Unmarshal(data, &cc); err != nil {
		return nil, err
	}
	return &cc, nil
}

func hasClientConfig() bool {
	_, err := os.Stat(filepath.Join(clientDir(), "client.yaml"))
	return err == nil
}

func hasServerConfig() bool {
	_, err := os.Stat(cfgPath())
	return err == nil
}

func serverAddr() (string, error) {
	if cc, err := loadClientConfig(); err == nil && cc.ServerAddress != "" {
		return cc.ServerAddress, nil
	}
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

	if err := signIfConfigured(req); err != nil {
		return nil, fmt.Errorf("signing request: %w", err)
	}

	return apiClient.Do(req)
}

func signIfConfigured(req *http.Request) error {
	dir := clientDir()
	cc, err := loadClientConfig()
	if err != nil {
		return nil
	}
	priv, err := auth.LoadPrivateKey(dir)
	if err != nil {
		return nil
	}
	return auth.SignRequest(req, priv, cc.KeyID)
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
