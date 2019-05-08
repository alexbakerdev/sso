package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/kelseyhightower/envconfig"

	log "github.com/buzzfeed/sso/internal/pkg/logging"
	"github.com/buzzfeed/sso/internal/proxy"
)

func init() {
	log.SetServiceName("sso-proxy")
}

func main() {
	logger := log.NewLogEntry()

	opts := proxy.NewOptions()
	err := envconfig.Process("", opts)
	if err != nil {
		logger.Error(err, "error parsing env vars into options")
		os.Exit(1)
	}

	err = opts.Validate()
	if err != nil {
		logger.Error(err, "error validing options")
		os.Exit(1)
	}

	ssoProxy, err := proxy.New(opts)
	if err != nil {
		logger.Error(err, "error creating sso proxy")
		os.Exit(1)
	}

	loggingHandler := proxy.NewLoggingHandler(os.Stdout,
		ssoProxy,
		opts.RequestLogging,
		opts.StatsdClient,
	)

	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", opts.Port),
		ReadTimeout:  opts.TCPReadTimeout,
		WriteTimeout: opts.TCPWriteTimeout,
		Handler:      loggingHandler,
	}

	logger.Fatal(s.ListenAndServe())
}
