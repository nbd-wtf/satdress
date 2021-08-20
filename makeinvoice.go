package main

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/fiatjaf/makeinvoice"
	"github.com/tidwall/sjson"
)

func makeMetadata(params *Params) string {
	metadata, _ := sjson.Set("[]", "0.0", "text/identifier")
	metadata, _ = sjson.Set(metadata, "0.1", params.Name+"@"+s.Domain)

	metadata, _ = sjson.Set(metadata, "1.0", "text/plain")
	metadata, _ = sjson.Set(metadata, "1.1", "Satoshis to "+params.Name+"@"+s.Domain+".")

	// TODO support image, custom description

	return metadata
}

func makeInvoice(params *Params, msat int) (bolt11 string, err error) {
	// description_hash
	h := sha256.Sum256([]byte(makeMetadata(params)))

	// prepare params
	var backend makeinvoice.BackendParams
	switch params.Kind {
	case "sparko":
		backend = makeinvoice.SparkoParams{
			Host: params.Host,
			Key:  params.Key,
		}
	case "lnd":
		backend = makeinvoice.LNDParams{
			Host:     params.Host,
			Macaroon: params.Key,
		}
	case "lnbits":
		backend = makeinvoice.LNBitsParams{
			Host: params.Host,
			Key:  params.Key,
		}
	case "lnpay":
		backend = makeinvoice.LNPayParams{
			PublicAccessKey:  params.Pak,
			WalletInvoiceKey: params.Waki,
		}
	}

	log.Debug().Int("msatoshi", msat).
		Interface("backend", backend).
		Str("description_hash", hex.EncodeToString(h[:])).
		Msg("generating invoice")

	// actually generate the invoice
	return makeinvoice.MakeInvoice(makeinvoice.Params{
		Msatoshi:        int64(msat),
		DescriptionHash: h[:],
		Backend:         backend,

		Label: s.Domain + "/" + strconv.FormatInt(time.Now().Unix(), 16),
	})
}
