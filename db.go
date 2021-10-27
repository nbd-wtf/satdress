package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cockroachdb/pebble"
)

type Params struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Host        string `json:"host"`
	Key         string `json:"key"`
	Pak         string `json:"pak"`
	Waki        string `json:"waki"`
	Pin         string `json:"pin"`
	MinSendable string `json:"minSendable"`
	MaxSendable string `json:"maxSendable"`
}

func SaveName(
	name string,
	params *Params,
	providedPin string,
) (pin string, inv string, err error) {
	name = strings.ToLower(name)
	key := []byte(name)

	pin = ComputePIN(name)

	if _, closer, err := db.Get(key); err == nil {
		defer closer.Close()
		if pin != providedPin {
			return "", "", errors.New("name already exists! must provide pin")
		}
	}
	if err != nil {
		return "", "", errors.New("that name does not exist")
	}

	params.Name = name

	// check if the given data works
	if inv, err = makeInvoice(params, 1000, &pin); err != nil {
		return "", "", fmt.Errorf("couldn't make an invoice with the given data: %w", err)
	}

	// save it
	data, _ := json.Marshal(params)
	if err := db.Set(key, data, pebble.Sync); err != nil {
		return "", "", err
	}

	return pin, inv, nil
}

func GetName(name string) (*Params, error) {
	name = strings.ToLower(name)

	val, closer, err := db.Get([]byte(name))
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	var params Params
	if err := json.Unmarshal(val, &params); err != nil {
		return nil, err
	}

	params.Name = name
	return &params, nil
}

func DeleteName(name string) error {
	name = strings.ToLower(name)
	key := []byte(name)

	if err := db.Delete(key, pebble.Sync); err != nil {
		return err
	}

	return nil
}

func ComputePIN(name string) string {
	name = strings.ToLower(name)
	mac := hmac.New(sha256.New, []byte(s.Secret))
	mac.Write([]byte(name + "@" + s.Domain))
	return hex.EncodeToString(mac.Sum(nil))
}
