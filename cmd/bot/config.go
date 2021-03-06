package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

type endpoint struct {
	ListenPath         string `json:"listen_path"`          // the path excluding domain to listen to, the good choice is "/your-telegram-bot-token"
	ListenAddress      string `json:"listen_address"`       // the address to listen to
	WebhookDomain      string `json:"webhook_domain"`       // the domain listening to the webhook
	CertificatePath    string `json:"certificate_path"`     // a path to your certificate, it is used to setup a webook and to setup this HTTP server
	CertificateKeyPath string `json:"certificate_key_path"` // your certificate key, omit if under a proxy
	BotToken           string `json:"bot_token"`            // your telegram bot token
	Translation        string `json:"translation"`          // translation strings
}

type coinPaymentsConfig struct {
	SubscriptionPacket string   `json:"subscription_packet"` // subscription packet, format "15/10" meaning 15 USD for 10 models
	Currencies         []string `json:"currencies"`          // CoinPayments currencies to buy a subscription with
	PublicKey          string   `json:"public_key"`          // CoinPayments public key
	PrivateKey         string   `json:"private_key"`         // CoinPayments private key
	IPNListenURL       string   `json:"ipn_listen_url"`      // CoinPayments IPN payment status notification listen URL
	IPNListenAddress   string   `json:"ipn_listen_address"`  // CoinPayments IPN payment status notification listen address
	IPNSecret          string   `json:"ipn_secret"`          // CoinPayments IPN secret

	subscriptionPacketPrice       int
	subscriptionPacketModelNumber int
}

type config struct {
	Website                     string              `json:"website"`                        // one of the following strings: "bongacams", "stripchat", "chaturbate"
	PeriodSeconds               int                 `json:"period_seconds"`                 // the period of querying models statuses
	MaxModels                   int                 `json:"max_models"`                     // maximum models per user
	TimeoutSeconds              int                 `json:"timeout_seconds"`                // HTTP timeout
	AdminID                     int64               `json:"admin_id"`                       // admin telegram ID
	AdminEndpoint               string              `json:"admin_endpoint"`                 // admin endpoint
	DBPath                      string              `json:"db_path"`                        // path to the database
	NotFoundThreshold           int                 `json:"not_found_threshold"`            // remove a model after she is reported as not found this number of times
	BlockThreshold              int                 `json:"block_threshold"`                // do not send a message to the user after being blocked by him this number of times
	Debug                       bool                `json:"debug"`                          // debug mode
	IntervalMs                  int                 `json:"interval_ms"`                    // queries interval per IP address for rate limited access
	SourceIPAddresses           []string            `json:"source_ip_addresses"`            // source IP addresses for rate limited access
	DangerousErrorRate          string              `json:"dangerous_error_rate"`           // dangerous error rate, warn admin if it is reached, format "1000/10000"
	EnableCookies               bool                `json:"enable_cookies"`                 // enable cookies, it can be useful to mitigate rate limits
	Headers                     [][2]string         `json:"headers"`                        // HTTP headers to make queries with
	StatPassword                string              `json:"stat_password"`                  // password for statistics
	ErrorReportingPeriodMinutes int                 `json:"error_reporting_period_minutes"` // the period of the error reports
	Endpoints                   map[string]endpoint `json:"endpoints"`                      // the endpoints by simple name, used for the support of the bots in different languages accessing the same database
	CoinPayments                *coinPaymentsConfig `json:"coin_payments"`                  // CoinPayments integration
	HeavyUserRemainder          int                 `json:"heavy_user_remainder"`           // the maximum remainder of models to treat an user as heavy
	MailHost                    string              `json:"mail_host"`                      // the hostname for email
	MailListenAddress           string              `json:"mail_listen_address"`            // the address to listen to incoming mail

	errorThreshold   int
	errorDenominator int
}

var fractionRegexp = regexp.MustCompile(`^(\d+)/(\d+)$`)

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
	for _, x := range cfg.Endpoints {
		if x.ListenAddress == "" {
			return errors.New("configure listen_address")
		}
		if x.ListenPath == "" {
			return errors.New("configure listen_path")
		}
		if x.BotToken == "" {
			return errors.New("configure bot_token")
		}
		if x.Translation == "" {
			return errors.New("configure translation")
		}
	}
	if _, found := cfg.Endpoints[cfg.AdminEndpoint]; !found {
		return errors.New("configure admin_endpoint")
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
	if cfg.NotFoundThreshold == 0 {
		return errors.New("configure not_found_threshold")
	}
	if cfg.BlockThreshold == 0 {
		return errors.New("configure block_threshold")
	}
	if cfg.Website == "" {
		return errors.New("configure website")
	}
	if cfg.StatPassword == "" {
		return errors.New("configure stat_password")
	}
	if cfg.ErrorReportingPeriodMinutes == 0 {
		return errors.New("configure error_reporting_period_minutes")
	}
	if cfg.HeavyUserRemainder == 0 {
		return errors.New("configure heavy_user_remainder")
	}
	if cfg.MailHost == "" {
		return errors.New("configure mail_host")
	}
	if cfg.MailListenAddress == "" {
		return errors.New("configure mail_listen_address")
	}

	if m := fractionRegexp.FindStringSubmatch(cfg.DangerousErrorRate); len(m) == 3 {
		errorThreshold, err := strconv.ParseInt(m[1], 10, 0)
		if err != nil {
			return err
		}

		errorDenominator, err := strconv.ParseInt(m[2], 10, 0)
		if err != nil {
			return err
		}

		if errorDenominator == 0 {
			return errors.New(`configure dangerous_errors_rate as "x/y", where y > 0`)
		}

		cfg.errorThreshold = int(errorThreshold)
		cfg.errorDenominator = int(errorDenominator)
	} else {
		return errors.New("configure dangerous_error_rate")
	}

	if cfg.CoinPayments != nil {
		if err := checkCoinPaymentsConfig(cfg.CoinPayments); err != nil {
			return err
		}
	}

	return nil
}

func checkCoinPaymentsConfig(cfg *coinPaymentsConfig) error {
	if len(cfg.Currencies) == 0 {
		return errors.New("configure currencies")
	}
	if cfg.PublicKey == "" {
		return errors.New("configure public_key")
	}
	if cfg.PrivateKey == "" {
		return errors.New("configure private_key")
	}
	if cfg.IPNListenURL == "" {
		return errors.New("configure ipn_path")
	}
	if cfg.IPNListenAddress == "" {
		return errors.New("configure ipn_path")
	}
	if cfg.IPNSecret == "" {
		return errors.New("configure ipn_secret")
	}

	if m := fractionRegexp.FindStringSubmatch(cfg.SubscriptionPacket); len(m) == 3 {
		subscriptionPacketModelNumber, err := strconv.ParseInt(m[1], 10, 0)
		if err != nil {
			return err
		}

		subscriptionPacketPrice, err := strconv.ParseInt(m[2], 10, 0)
		if err != nil {
			return err
		}

		if subscriptionPacketModelNumber == 0 || subscriptionPacketPrice == 0 {
			return errors.New("invalid subscription packet")
		}

		cfg.subscriptionPacketPrice = int(subscriptionPacketPrice)
		cfg.subscriptionPacketModelNumber = int(subscriptionPacketModelNumber)
	} else {
		return errors.New("configure subscription_packet")
	}

	return nil
}
