package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/fiatjaf/go-lnurl"
	"github.com/gorilla/mux"
)

func handleLNURL(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["user"]

	domains := getDomains(s.Domain)
	domain := ""

	if len(domains) == 1 {
		domain = domains[0]
	} else {
		hostname := r.URL.Host
		if hostname == "" {
			hostname = r.Host
		}

		for _, one := range getDomains(s.Domain) {
			if strings.Contains(hostname, one) {
				domain = one
				break
			}
		}
		if domain == "" {
			json.NewEncoder(w).Encode(lnurl.ErrorResponse("incorrect domain"))
			return
		}
	}

	params, err := GetName(username, domain)
	if err != nil {
		log.Error().Err(err).Str("name", username).Str("domain", domain).Msg("failed to get name")
		json.NewEncoder(w).Encode(lnurl.ErrorResponse(fmt.Sprintf(
			"failed to get name %s@%s", username, domain)))
		return
	}

	log.Info().Str("username", username).Str("domain", domain).Msg("got lnurl request")

	if amount := r.URL.Query().Get("amount"); amount == "" {
		// check if the receiver accepts comments
		var commentLength int64 = 0
		// TODO: support webhook comments

		// convert configured sendable amounts to integer
		minSendable, err := strconv.ParseInt(params.MinSendable, 10, 64)
		// set defaults
		if err != nil {
			minSendable = 1000
		}
		maxSendable, err := strconv.ParseInt(params.MaxSendable, 10, 64)
		if err != nil {
			maxSendable = 1000000000
		}

		json.NewEncoder(w).Encode(lnurl.LNURLPayResponse1{
			LNURLResponse:   lnurl.LNURLResponse{Status: "OK"},
			Callback:        fmt.Sprintf("https://%s/.well-known/lnurlp/%s", domain, username),
			MinSendable:     minSendable,
			MaxSendable:     maxSendable,
			EncodedMetadata: makeMetadata(params),
			CommentAllowed:  commentLength,
			Tag:             "payRequest",
		})

	} else {
		msat, err := strconv.Atoi(amount)
		if err != nil {
			json.NewEncoder(w).Encode(lnurl.ErrorResponse("amount is not integer"))
			return
		}

		bolt11, err := makeInvoice(params, msat, nil)
		if err != nil {
			json.NewEncoder(w).Encode(
				lnurl.ErrorResponse("failed to create invoice: " + err.Error()))
			return
		}

		json.NewEncoder(w).Encode(lnurl.LNURLPayResponse2{
			LNURLResponse: lnurl.LNURLResponse{Status: "OK"},
			PR:            bolt11,
			Routes:        make([][]lnurl.RouteInfo, 0),
			Disposable:    lnurl.FALSE,
			SuccessAction: lnurl.Action("Payment received!", ""),
		})

		// send webhook
		go func() {
			// TODO
		}()
	}
}
