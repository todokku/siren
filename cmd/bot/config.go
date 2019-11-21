package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

type config struct {
	Website                     string   `json:"website"`                         // one of the following: bongacams, stripchat, chaturbate
	ListenPath                  string   `json:"listen_path"`                     // the path excluding domain to listen to, the good choice is /your-telegram-bot-token
	ListenAddress               string   `json:"listen_address"`                  // the address to listen to
	WebhookDomain               string   `json:"webhook_domain"`                  // domain listening webhook
	BotToken                    string   `json:"bot_token"`                       // your telegram bot token
	PeriodSeconds               int      `json:"period_seconds"`                  // the period of querying models statuses
	MaxModels                   int      `json:"max_models"`                      // maximum models per user
	TimeoutSeconds              int      `json:"timeout_seconds"`                 // HTTP timeout
	AdminID                     int64    `json:"admin_id"`                        // your telegram ID
	DBPath                      string   `json:"db_path"`                         // path to database
	CertificatePath             string   `json:"certificate_path"`                // a path to your certificate
	CertificateKeyPath          string   `json:"certificate_key_path"`            // your key, omit if under a proxy
	NotFoundThreshold           int      `json:"not_found_threshold"`             // remove a model after a failure to find her this number of times
	BlockThreshold              int      `json:"block_threshold"`                 // do not send a message to the user if we fail to do it due to blocking this number of times
	Translation                 string   `json:"translation"`                     // translation strings
	Debug                       bool     `json:"debug"`                           // debug mode
	IntervalMs                  int      `json:"interval_ms"`                     // queries interval for rate limited access
	SourceIPAddresses           []string `json:"source_ip_addresses"`             // source IP address to use in queries
	DangerousErrorRateInPercent int      `json:"dangerous_error_rate_in_percent"` // dangerous error rate, warn admin if it is reached
	EnableCookies               bool     `json:"enable_cookies"`                  // enable cookies, it can be useful to mitigate rate limits
	UserAgent                   string   `json:"user_agent"`                      // user agent to make queries with
}

func readConfig(path string) *config {
	file, err := os.Open(filepath.Clean(path))
	checkErr(err)
	defer func() { checkErr(file.Close()) }()
	return parseConfig(file)
}

func parseConfig(r io.Reader) *config {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	cfg := &config{}
	err := decoder.Decode(cfg)
	checkErr(err)
	checkErr(checkConfig(cfg))
	if len(cfg.SourceIPAddresses) == 0 {
		cfg.SourceIPAddresses = append(cfg.SourceIPAddresses, "")
	}
	return cfg
}

func checkConfig(cfg *config) error {
	for _, x := range cfg.SourceIPAddresses {
		if net.ParseIP(x) == nil {
			return fmt.Errorf("cannot parse sourece IP address %s", x)
		}
	}
	if cfg.ListenPath == "" {
		return errors.New("configure listen_path")
	}
	if cfg.BotToken == "" {
		return errors.New("configure bot_token")
	}
	if cfg.PeriodSeconds == 0 {
		return errors.New("configure period_seconds")
	}
	if cfg.MaxModels == 0 {
		return errors.New("configure max_models")
	}
	if cfg.TimeoutSeconds == 0 {
		return errors.New("configure timeout_seconds")
	}
	if cfg.AdminID == 0 {
		return errors.New("configure admin_id")
	}
	if cfg.DBPath == "" {
		return errors.New("configure db_path")
	}
	if cfg.ListenAddress == "" {
		return errors.New("configure listen_address")
	}
	if cfg.NotFoundThreshold == 0 {
		return errors.New("configure not_found_threshold")
	}
	if cfg.BlockThreshold == 0 {
		return errors.New("configure block_threshold")
	}
	if cfg.Website == "" {
		return errors.New("configure website")
	}
	if cfg.Translation == "" {
		return errors.New("configure translation")
	}
	if cfg.DangerousErrorRateInPercent == 0 {
		return errors.New("configure dangerous_error_rate_in_percent")
	}
	return nil
}
