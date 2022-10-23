package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var lnproxyClient = &http.Client{}

func wrapInvoice(bolt11 string, msat, routing_msat int) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s?routing_msat=%d", s.LnproxyURL, bolt11, routing_msat), nil)
	if err != nil {
		return "", err
	}
	resp, err := lnproxyClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	buf := new(strings.Builder)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lnproxy error: %s", buf.String())
	}
	wbolt11 := strings.TrimSpace(buf.String())
	a, h, err := extractInvoiceDetails([]byte(bolt11))
	if err != nil {
		return "", err
	}
	wa, wh, err := extractInvoiceDetails([]byte(wbolt11))
	if err != nil {
		return "", err
	}
	if bytes.Compare(h, wh) != 0 {
		return "", fmt.Errorf("Wrapped payment hash does not match!")
	}
	if (a + routing_msat) != wa {
		return "", fmt.Errorf("Wrapped routing budget too high!")
	}
	return wbolt11, nil
}

var CharSet = []byte("qpzry9x8gf2tvdw0s3jn54khce6mua7l")

var validInvoice = regexp.MustCompile("^lnbc(?:[0-9]+[pnum])?1[qpzry9x8gf2tvdw0s3jn54khce6mua7l]+$")

func extractInvoiceDetails(invoice []byte) (int, []byte, error) {
	invoice = bytes.ToLower(invoice)
	pos := bytes.LastIndexByte(invoice, byte('1'))
	if pos == -1 || !validInvoice.Match(invoice) {
		return 0, nil, fmt.Errorf("Invalid invoice")
	}

	var msat int
	var err error
	if pos > 4 {
		msat, err = strconv.Atoi(string(invoice[4 : pos-1]))
		if err != nil {
			return 0, nil, err
		}
		switch invoice[pos-1] {
		case byte('p'):
			msat = msat / 10
		case byte('n'):
			msat = msat * 100
		case byte('u'):
			msat = msat * 100_000
		case byte('m'):
			msat = msat * 100_000_000
		}
	}
	for i := pos + 8; i < len(invoice); {
		if bytes.Compare(invoice[i:i+3], []byte("pp5")) == 0 {
			return msat, invoice[i+1+2 : i+1+2+52], nil
		}
		i += 3 + bytes.Index(CharSet, invoice[i+1:i+2])*32 + bytes.Index(CharSet, invoice[i+2:i+3])
	}
	return 0, nil, fmt.Errorf("No 'p' tag")
}
