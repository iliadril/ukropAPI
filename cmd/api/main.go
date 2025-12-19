package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"api.ukrop.pl/internal/data"
	"api.ukrop.pl/internal/mailer"
	"api.ukrop.pl/internal/spotify"
	"api.ukrop.pl/internal/vcs"
	"api.ukrop.pl/internal/youtube"
	_ "github.com/lib/pq"
)

var (
	version = vcs.Version()
)

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	yt struct {
		apiKey     string
		maxResults int
	}
	sp struct {
		clientID     string
		clientSecret string
		maxResults   int
	}
	cors struct {
		trustedOrigins []string
	}
}

type application struct {
	config  config
	logger  *slog.Logger
	models  data.Models
	mailer  *mailer.Mailer
	youtube *youtube.Client
	spotify *spotify.Client
	wg      sync.WaitGroup
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", "", "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 20, "PostgreSQL max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Ukrop <no-reply@ukrop.pl>", "SMTP sender")

	flag.StringVar(&cfg.yt.apiKey, "yt-api-key", os.Getenv("YOUTUBE_API_KEY"), "Api key for Youtube Data")
	flag.IntVar(&cfg.yt.maxResults, "yt-max-resuts", 5, "Max queries returned by Youtube api at once")

	flag.StringVar(&cfg.sp.clientID, "sp-client-id", os.Getenv("SPOTIFY_CLIENT_ID"), "Client ID for Spotify")
	flag.StringVar(&cfg.sp.clientSecret, "sp-client-secret", os.Getenv("SPOTIFY_CLIENT_SECRET"), "Client Secret for Spotify")
	flag.IntVar(&cfg.sp.maxResults, "sp-max-resuts", 5, "Max queries returned by Spotify api at once")

	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separate)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		os.Exit(0)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connection pool established")

	m, err := mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	yt, err := youtube.New(cfg.yt.apiKey)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	sp, err := spotify.New(cfg.sp.clientID, cfg.sp.clientSecret)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// ======== EXPVAR ========
	expvar.NewString("version").Set(version)
	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))
	expvar.Publish("database", expvar.Func(func() any {
		return db.Stats()
	}))
	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))
	// ======== END EXPVAR ========

	app := &application{
		config:  cfg,
		logger:  logger,
		models:  data.NewModels(db),
		mailer:  m,
		youtube: yt,
		spotify: sp,
	}

	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
