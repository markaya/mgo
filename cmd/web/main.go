package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/markaya/meinappf/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

/* TODO:
0. HTML/CSS - responsive reboot.
1. Fix race conditions
2. Add Apartments/Bills
3. Add Books
	a. MD Viewer
*/

type config struct {
	addr      string
	debugMode bool
	dsn       string
	tlsPath   string
}

type application struct {
	errorLog       *log.Logger
	infoLog        *log.Logger
	users          models.UserModelInterface
	accounts       models.AccountModelInterface
	transactions   models.TransactionsModelInterface
	templateCache  map[string]*template.Template
	sessionManager *scs.SessionManager
	debugMode      bool
}

func main() {
	cfg := config{}
	flag.StringVar(&cfg.addr, "address", ":4000", "HTTP network addr")
	// NOTE: Not needed because I use FS embed
	// flag.StringVar(&cfg.staticDir, "static-dir", "./ui/static", "Path to static assets")
	flag.BoolVar(&cfg.debugMode, "debug", false, "Turn debug mode on.")
	flag.StringVar(&cfg.dsn, "dsn", "", "Sqlite db string")
	flag.StringVar(&cfg.tlsPath, "tls", "./tls", "Tls folder")

	flag.Parse()
	flag.Usage()

	if cfg.dsn == "" {
		cfg.dsn = os.Getenv("MGO_DATABASE_URL")
	}

	// NOTE: Loggers
	infoLog := log.New(os.Stdout, "[INFO]\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "[ERROR]\t", log.Ldate|log.Ltime|log.Lshortfile)

	// NOTE: Database
	db, err := openDB(cfg.dsn)
	if err != nil {
		errorLog.Printf("there was an error trying to open db with dsn:\"%s\", please prived dsn flag or $MGO_DATABASE_URL env variable", cfg.dsn)
		errorLog.Fatal(err)
	}
	defer db.Close()

	// NOTE: Template cahce
	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}

	// NOTE: Session manager
	sessionManager := scs.New()
	sessionManager.Store = sqlite3store.New(db)
	sessionManager.Lifetime = 12 * time.Hour
	sessionManager.Cookie.Secure = true

	// NOTE: Application
	app := &application{
		errorLog:       errorLog,
		infoLog:        infoLog,
		users:          &models.UserModel{DB: db},
		accounts:       &models.AccountModel{DB: db},
		transactions:   &models.TransactionModel{DB: db},
		templateCache:  templateCache,
		sessionManager: sessionManager,
		debugMode:      cfg.debugMode,
	}

	// NOTE: TLS
	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		// You can set tls version here as well
		// NOTE: Enforce this so that TLS1.3 be used which enforces SameSite cookies.
		MinVersion: tls.VersionTLS13,
		// MaxVersion:       tls.VersionTLS13,
		// And also Cipher suits, to only use modern ones
		// CipherSuites: []uint16{
		// tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
		// etc..
		// }
	}

	srv := &http.Server{
		Addr:         cfg.addr,
		ErrorLog:     errorLog,
		Handler:      app.routes(),
		TLSConfig:    tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		// Limit max header length
		// MaxHeaderBytes:
	}

	infoLog.Printf("Starting server on %s\n", cfg.addr)
	cert := fmt.Sprintf("%s/cert.pem", cfg.tlsPath)
	key := fmt.Sprintf("%s/key.pem", cfg.tlsPath)
	err = srv.ListenAndServeTLS(cert, key)
	errorLog.Fatal(err)
}

func openDB(dsn string) (*sql.DB, error) {
	filePath := extractFilePath(dsn)
	if filePath == "" {
		return nil, fmt.Errorf("invalid DSN: file path not found")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database file does not exist: %s", filePath)
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, err
}

func extractFilePath(dsn string) string {
	if len(dsn) >= 5 && dsn[:5] == "file:" {
		dsn = dsn[5:]
	}
	for i, char := range dsn {
		if char == '?' {
			return dsn[:i]
		}
	}
	return dsn
}
