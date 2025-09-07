package config

import (
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	JWTSecret               string
	JWTUser                 string
	JWTPassword             string
	SearchTimeout           time.Duration
	CacheTTL                time.Duration
	TLSCertFile             string
	TLSKeyFile              string
	AmadeusURL              string
	AmadeusClientId         string
	AmadeusClientSSecret    string
	DuffelHost              string
	DuffelToken             string
	RapidBookingHost        string
	RapidBookingRapidApiKey string
}

func Load() *Config {
	v := viper.New()

	v.SetDefault("auth_user", "demo")
	v.SetDefault("auth_pass", "demo123")
	v.SetDefault("search_timeout", "10s")
	v.SetDefault("cache_ttl", "30s")

	v.SetDefault("amadeus_url", "https://test.api.amadeus.com")
	v.SetDefault("duffel_host", "https://api.duffel.com")
	v.SetDefault("rapid_booking_host", "booking-com15.p.rapidapi.com")

	if path := os.Getenv("FLIGHTS_CONFIG"); path != "" {
		v.SetConfigFile(path)
	} else {
		// Fallback to conventional locations for local dev
		v.SetConfigName("config")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/flights") // add the container path
	}

	if err := v.ReadInConfig(); err != nil {
		log.Printf("no config file found, using defaults + env vars: %v", err)
	}

	v.AutomaticEnv()

	to, err := time.ParseDuration(v.GetString("search_timeout"))
	if err != nil {
		log.Fatalf("bad search_timeout: %v", err)
	}
	ct, err := time.ParseDuration(v.GetString("cache_ttl"))
	if err != nil {
		log.Fatalf("bad cache_ttl: %v", err)
	}

	return &Config{
		JWTSecret:               v.GetString("jwt_secret"),
		JWTUser:                 v.GetString("auth_user"),
		JWTPassword:             v.GetString("auth_pass"),
		SearchTimeout:           to,
		CacheTTL:                ct,
		TLSCertFile:             os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:              os.Getenv("TLS_KEY_FILE"),
		AmadeusURL:              v.GetString("amadeus_url"),
		AmadeusClientId:         v.GetString("amadeus_clientid"),
		AmadeusClientSSecret:    v.GetString("amadeus_clientsecret"),
		DuffelHost:              v.GetString("duffel_host"),
		DuffelToken:             v.GetString("duffel_token"),
		RapidBookingHost:        v.GetString("rapid_booking_host"),
		RapidBookingRapidApiKey: v.GetString("rapid_booking_rapidapikey"),
	}
}
