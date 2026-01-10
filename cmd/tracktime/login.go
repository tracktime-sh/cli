package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"github.com/tracktime-sh/cli/internal/config"
)

const (
	callbackPort    = 19876
	callbackTimeout = 5 * time.Minute
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with tracktime.sh",
	Long:  `Opens your browser to authenticate with tracktime.sh and stores the API key locally.`,
	RunE:  runLogin,
}

func runLogin(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	if cfg.IsConfigured() {
		fmt.Println("You are already logged in.")
		fmt.Println("Run 'tracktime logout' first if you want to switch accounts.")
		return nil
	}

	state, err := generateState()
	if err != nil {
		fmt.Printf("Error generating state: %v\n", err)
		os.Exit(1)
	}

	keyChan := make(chan string, 1)
	errChan := make(chan error, 1)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", callbackPort))
	if err != nil {
		fmt.Printf("Error starting callback server: %v\n", err)
		fmt.Println("Another process might be using port 19876. Please try again.")
		os.Exit(1)
	}

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			key := r.URL.Query().Get("key")
			returnedState := r.URL.Query().Get("state")

			if returnedState != state {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, htmlResponse("Authentication Failed", "Invalid state parameter. Please try again.", false))
				errChan <- fmt.Errorf("state mismatch")
				return
			}

			if key == "" {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, htmlResponse("Authentication Failed", "No API key received. Please try again.", false))
				errChan <- fmt.Errorf("no API key received")
				return
			}

			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, htmlResponse("Authentication Successful", "You can close this window and return to the terminal.", true))
			keyChan <- key
		}),
	}

	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	authURL := fmt.Sprintf("%s/cli-auth?port=%d&state=%s", config.WebURL, callbackPort, state)

	fmt.Println("Opening browser to authenticate...")
	fmt.Printf("If the browser doesn't open, visit: %s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser automatically.\n")
		fmt.Printf("Please open this URL manually: %s\n\n", authURL)
	}

	fmt.Println("Waiting for authentication...")

	ctx, cancel := context.WithTimeout(context.Background(), callbackTimeout)
	defer cancel()

	select {
	case key := <-keyChan:
		server.Shutdown(context.Background())
		cfg.APIKey = key
		if err := config.Save(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\nLogged in successfully!")
		return nil

	case err := <-errChan:
		server.Shutdown(context.Background())
		fmt.Printf("\nAuthentication failed: %v\n", err)
		os.Exit(1)
		return nil

	case <-ctx.Done():
		server.Shutdown(context.Background())
		fmt.Println("\nAuthentication timed out. Please try again.")
		os.Exit(1)
		return nil
	}
}

func generateState() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

func htmlResponse(title, message string, success bool) string {
	color := "#ef4444"
	if success {
		color = "#22c55e"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s - tracktime</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: #111;
            color: #fff;
        }
        .container {
            text-align: center;
            padding: 2rem;
        }
        h1 {
            color: %s;
            margin-bottom: 1rem;
        }
        p {
            color: #999;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>%s</h1>
        <p>%s</p>
    </div>
</body>
</html>`, title, color, title, message)
}
