package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/ardanlabs/conf"
	"gitlab.com/enrico204/http-telescope/service/telescope"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
)

// These two variables are modified at build-time

// AppVersion contains the app version (tag + commit after tag + current git ref)
var AppVersion = "devel"

// BuildDate contains the timestamp of the build
var BuildDate = "n/a"

// Main function
func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error: ", err)
		os.Exit(1)
	}
}

func run() error {
	// Load Configuration and defaults
	cfg, err := loadConfiguration()
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			return nil
		}
		return err
	}

	// Init logging
	logger, err := newLogger(cfg.Log)
	if err != nil {
		return err
	}

	// Print the build version for our logs. Also expose it under /debug/vars.
	logger.Infof("application initializing, version %q (%s)", AppVersion, BuildDate)

	// Print out the configuration
	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("generating config for output: %w", err)
	}
	logger.Debugf("config: %v", out)

	// Start proxy server
	logger.Info("initializing reverse proxy")

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	if cfg.Web.Upstream == "" {
		logger.Error("error: empty upstream URL")
		return fmt.Errorf("empty upstream URL")
	}
	// Start the reverse proxy
	remote, err := url.Parse(cfg.Web.Upstream)
	if err != nil {
		logger.WithError(err).Error("error parsing upstream URL")
		return fmt.Errorf("parsing upstream URL: %w", err)
	}

	tscope, err := telescope.New(remote, telescope.Options{StoreBody: cfg.StoreBody, DisableCaching: cfg.DisableCaching})
	if err != nil {
		logger.WithError(err).Error("error creating Telescope instance")
		return fmt.Errorf("creating Telescope instance: %w", err)
	}
	reverseProxy := &http.Server{Addr: cfg.Web.Listen, Handler: tscope}

	// Start the service listening for requests.
	go func() {
		logger.Infof("Reverse proxy listening on %s", reverseProxy.Addr)
		serverErrors <- reverseProxy.ListenAndServe()
		logger.Infof("stopping reverse proxy")
	}()

	// Start web dashboard
	logger.Info("initializing separate web dashboard")
	http.HandleFunc("/", tscope.ServeWebDashboard)
	go func() {
		logger.Infof("web dashboard %s", cfg.Web.UI)
		logger.Infof("web dashboard closed: %v", http.ListenAndServe(cfg.Web.UI, http.DefaultServeMux))
	}()

	// =========================================================================
	// Shutdown

	// Blocking main and waiting for shutdown signal or POSIX signals
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		logger.Infof("signal %v received, start shutdown", sig)

		// Asking proxy server to shutdown and load shed.
		err := reverseProxy.Close()
		if err != nil {
			logger.WithError(err).Warning("graceful shutdown of reverse proxy error")
		}

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()

		// Asking listener to shutdown and load shed.
		err = reverseProxy.Shutdown(ctx)
		if err != nil {
			logger.WithError(err).Warning("error during graceful shutdown of HTTP server")
			err = reverseProxy.Close()
		}

		// Log the status of this shutdown.
		switch {
		case sig == syscall.SIGSTOP:
			return errors.New("integrity issue caused shutdown")
		case err != nil:
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}
