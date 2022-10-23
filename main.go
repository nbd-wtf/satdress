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
	"github.com/rs/cors"
	"github.com/rs/zerolog"
)

type Settings struct {
	Host   string `envconfig:"HOST" default:"0.0.0.0"`
	Port   string `envconfig:"PORT" required:"true"`
	Domain string `envconfig:"DOMAIN" required:"true"`
	// GlobalUsers means that user@ part is globally unique across all domains
	// WARNING: if you toggle this existing users won't work anymore for safety reasons!
	GlobalUsers   bool   `envconfig:"GLOBAL_USERS" required:"false" default:false`
	Secret        string `envconfig:"SECRET" required:"true"`
	SiteOwnerName string `envconfig:"SITE_OWNER_NAME" required:"true"`
	SiteOwnerURL  string `envconfig:"SITE_OWNER_URL" required:"true"`
	SiteName      string `envconfig:"SITE_NAME" required:"true"`

	ForceMigrate bool   `envconfig:"FORCE_MIGRATE" required:"false" default:false`
	TorProxyURL  string `envconfig:"TOR_PROXY_URL"`
}

var (
	s      Settings
	db     *pebble.DB
	router = mux.NewRouter()
	log    = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
)

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

	// increase default makeinvoice client timeout because people are using tor
	makeinvoice.Client = &http.Client{Timeout: 25 * time.Second}

	s.Domain = strings.ToLower(s.Domain)

	if s.TorProxyURL != "" {
		makeinvoice.TorProxyURL = s.TorProxyURL
	}

	dbName := fmt.Sprintf("%v-multiple.db", s.SiteName)
	if _, err := os.Stat(dbName); os.IsNotExist(err) || s.ForceMigrate {
		for _, one := range getDomains(s.Domain) {
			tryMigrate(one, dbName)
		}
	}

	db, err = pebble.Open(dbName, nil)
	if err != nil {
		log.Fatal().Err(err).Str("path", dbName).Msg("failed to open db.")
	}

	router.Path("/.well-known/lnurlp/{user}").Methods("GET").
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
			if name == "" || r.FormValue("kind") == "" {
				sendError(w, 500, "internal error")
				return
			}

			// might not get domain back
			domain := r.FormValue("domain")
			if domain == "" {
				if !strings.Contains(s.Domain, ",") {
					domain = s.Domain
				} else {
					sendError(w, 500, "internal error")
					return
				}
			}

			pin, inv, err := SaveName(name, domain, &Params{
				Kind:   r.FormValue("kind"),
				Host:   r.FormValue("host"),
				Key:    r.FormValue("key"),
				Pak:    r.FormValue("pak"),
				Waki:   r.FormValue("waki"),
				NodeId: r.FormValue("nodeid"),
				Rune:   r.FormValue("rune"),
			}, r.FormValue("pin"))
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}

			renderHTML(w, grabHTML, struct {
				PIN          string `json:"pin"`
				Invoice      string `json:"invoice"`
				Name         string `json:"name"`
				ActualDomain string `json:"actual_domain"`
			}{pin, inv, name, domain})
		},
	)

	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(authenticate)

	// unauthenticated
	api.HandleFunc("/claim", ClaimAddress).Methods("POST")

	// authenticated routes; X-Pin in header or in json request body
	api.HandleFunc("/users/{name}@{domain}", GetUser).Methods("GET")
	api.HandleFunc("/users/{name}@{domain}", UpdateUser).Methods("PUT")
	api.HandleFunc("/users/{name}@{domain}", DeleteUser).Methods("DELETE")

	srv := &http.Server{
		Handler:      cors.Default().Handler(router),
		Addr:         s.Host + ":" + s.Port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Debug().Str("addr", srv.Addr).Msg("listening")
	srv.ListenAndServe()
}

func getDomains(s string) []string {
	splitFn := func(c rune) bool {
		return c == ','
	}
	return strings.FieldsFunc(s, splitFn)
}
