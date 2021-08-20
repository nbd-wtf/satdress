package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cockroachdb/pebble"
)

type Params struct {
	Name string
	Kind string
	Host string
	Key  string
	Pak  string
	Waki string
}

func SaveName(name string, params *Params, providedPin string) (pin string, err error) {
	key := []byte(name + "@" + s.Domain)

	mac := hmac.New(sha256.New, []byte(s.Secret))
	mac.Write(key)
	pin = hex.EncodeToString(mac.Sum(nil))

	if _, closer, err := db.Get(key); err == nil {
		defer closer.Close()
		if pin != providedPin {
			return "", errors.New("name already exists! must provide pin.")
		}
	}

	// check if the given data works
	if _, err := makeInvoice(params, 1000); err != nil {
		return "", fmt.Errorf("couldn't make an invoice with the given data: %w", err)
	}

	// save it
	data, _ := json.Marshal(params)
	if err := db.Set(key, data, pebble.Sync); err != nil {
		return "", err
	}

	return pin, nil
}

func GetName(name string) (params *Params, err error) {
	val, closer, err := db.Get([]byte(name))
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	if err := json.Unmarshal(val, params); err != nil {
		return nil, err
	}

	params.Name = name
	return params, nil
}
