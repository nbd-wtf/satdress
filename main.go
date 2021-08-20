package main

import (
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cockroachdb/pebble"
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
}

var s Settings
var db *pebble.DB
var router = mux.NewRouter()
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})

//go:embed index.html
var html string

func main() {
	err := envconfig.Process("", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig.")
	}

	db, err = pebble.Open(s.Domain, nil)
	if err != nil {
		log.Fatal().Err(err).Str("path", s.Domain).Msg("failed to open db.")
	}

	router.Path("/.well-known/lnurlp/{username}").Methods("GET").
		HandlerFunc(handleLNURL)

	router.Path("/").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("content-type", "text/html")
			serverData, _ := json.Marshal(struct {
				Domain        string `json:"domain"`
				SiteOwnerName string `json:"siteOwnerName"`
				SiteOwnerURL  string `json:"siteOwnerURL"`
				SiteName      string `json:"siteName"`
			}{
				Domain:        s.Domain,
				SiteOwnerName: s.SiteOwnerName,
				SiteOwnerURL:  s.SiteOwnerURL,
				SiteName:      s.SiteName,
			})
			fmt.Fprint(w,
				strings.ReplaceAll(
					strings.ReplaceAll(
						html, "{} // REPLACED WITH SERVER DATA", string(serverData),
					),
					"Satdress", s.SiteName,
				),
			)
		},
	)

	router.Path("/grab").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			name := []byte(r.FormValue("name") + "@" + s.Domain)

			mac := hmac.New(sha256.New, []byte(s.Secret))
			mac.Write(name)
			pin := hex.EncodeToString(mac.Sum(nil))

			if _, closer, err := db.Get(name); err == nil {
				w.WriteHeader(401)
				fmt.Fprint(w,
					"name already exists! must provide pin (contact support).")
				return
			} else if err == nil {
				closer.Close()
			}

			params := Params{
				Kind: r.FormValue("kind"),
				Host: r.FormValue("host"),
				Key:  r.FormValue("key"),
				Pak:  r.FormValue("pak"),
				Waki: r.FormValue("waki"),
			}

			// check if the given data works
			if _, err := makeInvoice(params, 1000); err != nil {
				w.WriteHeader(400)
				fmt.Fprint(w, "couldn't make an invoice with the given data: "+err.Error())
				return
			}

			// save it
			data, _ := json.Marshal(params)
			if err := db.Set(name, data, pebble.Sync); err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, "error! "+err.Error())
				return
			}

			fmt.Fprintf(w,
				"name saved! this is your secret pin key for this name: %s",
				pin)
		},
	)

	srv := &http.Server{
		Handler:      router,
		Addr:         s.Host + ":" + s.Port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Debug().Str("addr", srv.Addr).Msg("listening")
	srv.ListenAndServe()
}
