package main

import (
	"crypto/tls"
	"cx1resultsviewer/internal/backend"
	"flag"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.TraceLevel)
	myformatter := &easy.Formatter{}
	myformatter.TimestampFormat = "2006-01-02 15:04:05.000"
	myformatter.LogFormat = "[%lvl%][%time%] %msg%\n"
	logger.SetFormatter(myformatter)
	// Use stderr so logs don't corrupt the MCP stdio transport on stdout.
	logger.SetOutput(os.Stdout)

	logger.Info("Starting")
	LogLevel := flag.String("log", "INFO", "Log level: TRACE, DEBUG, INFO, WARNING, ERROR, FATAL")
	Address := flag.String("address", "127.0.0.1:8080", "Listen address")
	_ = flag.String("proxy", "", "Proxy server to use connecting out to CheckmarxOne")

	proxy := getProxy()
	httpClient := &http.Client{}
	if proxy != "" {
		proxyURL, _ := url.Parse(proxy)
		transport := &http.Transport{}
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		httpClient.Transport = transport
		logger.Infof("Using proxy %s", proxy)
	}

	cx1client, err := Cx1ClientGo.NewClient(httpClient, logger)
	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}
	logger.Infof("Connected with %v", cx1client.String())

	switch strings.ToUpper(*LogLevel) {
	case "TRACE":
		logger.SetLevel(logrus.TraceLevel)
	case "DEBUG":
		logger.SetLevel(logrus.DebugLevel)
	case "INFO":
		logger.SetLevel(logrus.InfoLevel)
	case "WARNING":
		logger.SetLevel(logrus.WarnLevel)
	case "ERROR":
		logger.SetLevel(logrus.ErrorLevel)
	case "FATAL":
		logger.SetLevel(logrus.FatalLevel)
	}

	server := backend.NewServer(cx1client, logger, *Address)
	defer server.Shutdown()

	if err = server.Run(); err != nil {
		logger.Errorf("Failed while running server: %s", err)
	}

	logger.Info("Done!")
}

func getProxy() string {
	for i := 0; i < len(os.Args); i++ {
		// Match exact '-proxy' or '--proxy'
		if os.Args[i] == "-proxy" || os.Args[i] == "--proxy" {
			// Ensure there's a following value argument
			if i+1 < len(os.Args) {
				return os.Args[i+1]
			}
			break
		}
		// Alternatively catch inline assignments like -proxy=http://...
		if len(os.Args[i]) > 7 && os.Args[i][:7] == "-proxy=" {
			return os.Args[i][7:]
		}
		if len(os.Args[i]) > 8 && os.Args[i][:8] == "--proxy=" {
			return os.Args[i][8:]
		}
	}
	return ""
}
