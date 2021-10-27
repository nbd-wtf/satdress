package main

import (
	"embed"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/fiatjaf/makeinvoice"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

type Settings struct {
	Host          string `envconfig:"HOST" default:"0.0.0.0"`
	Port          string `envconfig:"PORT" required:"true"`
	Domain        string `envconfig:"DOMAIN" required:"true"`
	Secret        string `envconfig:"SECRET" required:"true"`
	SiteOwnerName string `envconfig:"SITE_OWNER_NAME" required:"true"`
	SiteOwnerURL  string `envconfig:"SITE_OWNER_URL" required:"true"`
	SiteName      string `envconfig:"SITE_NAME" required:"true"`

	TorProxyURL string `envconfig:"TOR_PROXY_URL"`
}

var s Settings
var db *pebble.DB
var router = mux.NewRouter()
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})

//go:embed index.html
var indexHTML string

//go:embed grab.html
var grabHTML string

//go:embed static
var static embed.FS

func main() {
	err := envconfig.Process("", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig.")
	}

	s.Domain = strings.ToLower(s.Domain)

	if s.TorProxyURL != "" {
		makeinvoice.TorProxyURL = s.TorProxyURL
	}

	db, err = pebble.Open(s.Domain, nil)
	if err != nil {
		log.Fatal().Err(err).Str("path", s.Domain).Msg("failed to open db.")
	}

	router.Path("/.well-known/lnurlp/{username}").Methods("GET").
		HandlerFunc(handleLNURL)

	router.Path("/").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			renderHTML(w, indexHTML, map[string]interface{}{})
		},
	)

	router.PathPrefix("/static/").Handler(http.FileServer(http.FS(static)))

	router.Path("/grab").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			name := r.FormValue("name")

			pin, inv, err := SaveName(name, &Params{
				Kind: r.FormValue("kind"),
				Host: r.FormValue("host"),
				Key:  r.FormValue("key"),
				Pak:  r.FormValue("pak"),
				Waki: r.FormValue("waki"),
			}, r.FormValue("pin"))
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}

			renderHTML(w, grabHTML, struct {
				PIN     string `json:"pin"`
				Invoice string `json:"invoice"`
				Name    string `json:"name"`
			}{pin, inv, name})
		},
	)

	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(authenticate)

	// unauthenticated
	api.HandleFunc("/claim", ClaimAddress).Methods("POST")

	// authenticated routes; X-Pin in header or in json request body
	api.HandleFunc("/users/{name}", GetUser).Methods("GET")
	api.HandleFunc("/users/{name}", UpdateUser).Methods("PUT")
	api.HandleFunc("/users/{name}", DeleteUser).Methods("DELETE")

	srv := &http.Server{
		Handler:      router,
		Addr:         s.Host + ":" + s.Port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Debug().Str("addr", srv.Addr).Msg("listening")
	srv.ListenAndServe()
}
