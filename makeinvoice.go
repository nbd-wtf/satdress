package main

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"time"

	"github.com/fiatjaf/makeinvoice"
	"github.com/tidwall/sjson"
)

func makeMetadata(params *Params) string {
	metadata, _ := sjson.Set("[]", "0.0", "text/identifier")
	metadata, _ = sjson.Set(metadata, "0.1", params.Name+"@"+params.Domain)

	metadata, _ = sjson.Set(metadata, "1.0", "text/plain")
	metadata, _ = sjson.Set(metadata, "1.1", "Satoshis to "+params.Name+"@"+params.Domain+".")

	// TODO support image, custom description

	return metadata
}

func makeInvoice(
	params *Params,
	msat int,
	pin *string,
) (bolt11 string, err error) {
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
	case "eclair":
		backend = makeinvoice.EclairParams{
			Host:     params.Host,
			Password: "",
		}
	}

	mip := makeinvoice.Params{
		Msatoshi: int64(msat),
		Backend:  backend,

		Label: s.Domain + "/" + strconv.FormatInt(time.Now().Unix(), 16),
	}

	if pin != nil {
		// use this as the description
		mip.Description = fmt.Sprintf("%s's PIN for '%s@%s' lightning address: %s", s.Domain, params.Name, s.Domain, *pin)
	} else {
		// make the lnurlpay description_hash
		h := sha256.Sum256([]byte(makeMetadata(params)))
		mip.DescriptionHash = h[:]
	}

	// actually generate the invoice
	bolt11, err = makeinvoice.MakeInvoice(mip)

	log.Debug().Int("msatoshi", msat).
		Interface("backend", backend).
		Str("bolt11", bolt11).Err(err).
		Msg("invoice generation")

	return bolt11, err
}
