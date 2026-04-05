package cli

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/christianmscott/overwatch/internal/auth"
	"github.com/christianmscott/overwatch/internal/config"
	"github.com/christianmscott/overwatch/pkg/spec"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up Overwatch (server, client, or cloud)",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Welcome to Overwatch!")
	fmt.Println()
	fmt.Println("How would you like to set up?")
	fmt.Println("  1. Setup a server     - run a self-hosted monitoring server")
	fmt.Println("  2. Connect to Cloud   - use Overwatch Cloud (coming soon)")
	fmt.Println("  3. Setup a client     - connect to an existing server")
	fmt.Println()
	fmt.Print("Select [1-3]: ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		return initServer()
	case "2":
		return initCloud()
	case "3":
		return initClient(reader)
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}
}

func initServer() error {
	path := cfgPath()
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists; remove it first or choose a different path with --config", path)
	}

	cfg := &spec.Config{}
	if err := yaml.Unmarshal([]byte(config.StarterConfig), cfg); err != nil {
		return fmt.Errorf("parsing starter config: %w", err)
	}

	joinToken, err := auth.GenerateJoinToken(cfg.Server.TokenAddress())
	if err != nil {
		return err
	}
	cfg.Server.JoinToken = joinToken

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("\nWrote server config to %s\n", path)
	fmt.Println()
	fmt.Println("==> Join token:", joinToken)
	fmt.Println("==> Share this token with clients to connect to this server.")
	fmt.Println()
	fmt.Println("Start the server with: overwatch serve")
	return nil
}

func initCloud() error {
	fmt.Println()
	fmt.Println("Overwatch Cloud is coming soon.")
	fmt.Println("Visit https://overwatch.dev to sign up for early access.")
	return nil
}

func initClient(reader *bufio.Reader) error {
	dir := clientDir()
	if _, err := os.Stat(filepath.Join(dir, "client.yaml")); err == nil {
		return fmt.Errorf("client already configured at %s; remove it to reconfigure", dir)
	}

	fmt.Println()
	fmt.Print("Enter join token (from server logs or config): ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	addr, _, err := auth.ParseJoinToken(token)
	if err != nil {
		return fmt.Errorf("invalid join token: %w", err)
	}

	fmt.Printf("Connecting to server at %s...\n", addr)

	pub, priv, err := auth.GenerateKeypair()
	if err != nil {
		return err
	}
	keyID := auth.KeyID(pub)

	if err := auth.SaveKeys(dir, pub, priv); err != nil {
		return err
	}

	serverURL := fmt.Sprintf("http://%s", addr)
	if err := sendJoinRequest(serverURL, token, pub); err != nil {
		return fmt.Errorf("join failed: %w", err)
	}

	clientCfg := spec.ClientConfig{
		ServerAddress: serverURL,
		KeyID:         keyID,
	}
	data, err := yaml.Marshal(clientCfg)
	if err != nil {
		return fmt.Errorf("marshalling client config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "client.yaml"), data, 0644); err != nil {
		return fmt.Errorf("writing client config: %w", err)
	}

	fmt.Println()
	fmt.Println("Client configured successfully.")
	fmt.Printf("  Key ID:  %s\n", keyID)
	fmt.Printf("  Server:  %s\n", serverURL)
	fmt.Printf("  Config:  %s\n", dir)
	fmt.Println()
	fmt.Println("Run 'overwatch status' to check the server.")
	return nil
}

func sendJoinRequest(serverURL, joinToken string, pub []byte) error {
	hostname, _ := os.Hostname()
	body, _ := json.Marshal(map[string]string{
		"join_token": joinToken,
		"public_key": base64.StdEncoding.EncodeToString(pub),
		"label":      hostname,
	})
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(serverURL+"/api/join", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("contacting server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var ae struct{ Error string }
		json.NewDecoder(resp.Body).Decode(&ae)
		if ae.Error != "" {
			return fmt.Errorf("server: %s", ae.Error)
		}
		return fmt.Errorf("server returned %s", resp.Status)
	}
	return nil
}

func clientDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), ".overwatch")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".overwatch")
}
