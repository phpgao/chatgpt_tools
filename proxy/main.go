package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

var (
	serverPort       string
	authorizationKey string
	targetAPI        string
	skipWord         string
	cert             string
	key              string
	signalCommand    string
)

func init() {
	flag.StringVar(&serverPort, "port", "8088", "HTTP server port")
	flag.StringVar(&authorizationKey, "apikey", "", "API key for authorization")
	flag.StringVar(&targetAPI, "target", "", "Target server URL")
	flag.StringVar(&skipWord, "skip", "q", "Word to skip")
	flag.StringVar(&cert, "cert", "", "Certificate")
	flag.StringVar(&key, "key", "", "Key")
	flag.StringVar(&signalCommand, "s", "", "Send 'stop' or 'reload' signal to the process")
	flag.Parse()
}

type RequestBody struct {
	Messages         []Messages `json:"messages"`
	Stream           bool       `json:"stream"`
	Model            string     `json:"model"`
	Temperature      float64    `json:"temperature"`
	PresencePenalty  int        `json:"presence_penalty"`
	FrequencyPenalty int        `json:"frequency_penalty"`
	TopP             int        `json:"top_p"`
}

type Messages struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func createProxyServer(targetURL *url.URL, apiKey string) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.Host = targetURL.Host
		req.URL.Path = targetURL.Path
		if req.Method != http.MethodOptions {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		// 读取原始body
		if req.Body != nil {
			originalBody, err := io.ReadAll(req.Body)
			if err != nil {
				log.Printf("Error reading body: %v", err)
				return
			}
			req.Body.Close()

			var bodyData RequestBody
			if err := json.Unmarshal(originalBody, &bodyData); err != nil {
				log.Printf("Error unmarshalling JSON: %v", err)
				return
			}
			// 移除skipWord
			bodyData.Model = strings.ReplaceAll(bodyData.Model, fmt.Sprintf("(%s)", skipWord), "")
			modifiedBody, err := json.Marshal(bodyData)
			if err != nil {
				log.Printf("Error marshalling JSON: %v", err)
				return
			}

			req.Body = io.NopCloser(bytes.NewReader(modifiedBody))
			req.ContentLength = int64(len(modifiedBody))
			req.Header.Set("Content-Length", fmt.Sprint(len(modifiedBody))) // 更新Content-Length头
		}
	}
	return proxy
}

func setupCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
}

var srv *http.Server

func startServer(serverPort string, handler http.Handler) {
	srv = &http.Server{
		Addr:    ":" + serverPort,
		Handler: handler,
	}

	log.Printf("Server is running on port %s", serverPort)
	var err error
	if cert != "" && key != "" {
		err = srv.ListenAndServeTLS(cert, key)
	} else {
		err = srv.ListenAndServe()
	}
	if !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Error starting server: %v", err)
	}
}

func newMux() *http.ServeMux {
	targetURL, err := url.Parse(targetAPI)
	if err != nil {
		log.Fatalf("Error parsing the target API URL: %v", err)
	}
	proxy := createProxyServer(targetURL, authorizationKey)
	mux := http.NewServeMux()
	mux.Handle("/v1/chat/completions", proxy)
	return mux
}

func shutdownServer(ctx context.Context) {
	// ... 关闭服务器的逻辑 ...
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down server: %v", err)
	} else {
		log.Println("Server has been gracefully shutdown")
	}
}

func sendSignal(signal os.Signal) {
	pid, err := getCurrentPID()
	if err != nil {
		fmt.Printf("Failed to get current process PID: %v\n", err)
		os.Exit(1)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("Failed to find process: %v\n", err)
		os.Exit(1)
	}

	err = process.Signal(signal)
	if err != nil {
		fmt.Printf("Failed to send signal: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully sent signal %v to process %d\n", signal, pid)
}

func getCurrentPID() (int, error) {
	//read pid from file
	f, err := os.Open("/var/run/gpt-proxy.pid")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func signalHandler(signalChan chan os.Signal, ctx context.Context) {
	for {
		sig := <-signalChan
		switch sig {
		case syscall.SIGHUP, syscall.SIGUSR1:
			log.Info("Received SIGUSR1 signal, gracefully shutting down current server...")
			shutdownServer(context.Background())
			log.Info("Starting new HTTP server...")
			go startServer(serverPort, newMux())
		case syscall.SIGINT, syscall.SIGTERM:
			log.Info("Received SIGTERM signal, gracefully shutting down server...")
			shutdownServer(ctx)
			os.Exit(0)
		}
	}
}

func writePIDToFile(pid int) {
	pidFilePath := "/var/run/gpt-proxy.pid"
	pidString := strconv.Itoa(pid)
	err := os.WriteFile(pidFilePath, []byte(pidString), 0644)
	if err != nil {
		log.WithError(err).Error("Error writing PID file")
	}
}

func main() {
	if signalCommand != "" {
		switch signalCommand {
		case "stop":
			sendSignal(syscall.SIGTERM)
		case "reload":
			sendSignal(syscall.SIGUSR1)
		default:
			fmt.Println("Invalid signal, available signals are 'stop' or 'reload'")
		}
		return
	}

	if authorizationKey == "" || targetAPI == "" {
		log.Fatal("API key or target API is not set, please provide them via -apikey and -target flags.")
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go signalHandler(signalChan, ctx)

	go startServer(serverPort, newMux())
	pid := os.Getpid()
	log.Infof("Server started, PID: %d", pid)
	writePIDToFile(pid)

	<-ctx.Done()
}
