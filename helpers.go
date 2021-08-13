package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"strconv"

	"github.com/nfnt/resize"
)

func base64ImageFromURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", errors.New("image returned status " + strconv.Itoa(resp.StatusCode))
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to decode image from %s: %w", url, err)
	}

	img = resize.Resize(160, 0, img, resize.NearestNeighbor)
	out := &bytes.Buffer{}
	err = jpeg.Encode(out, img, &jpeg.Options{60})
	if err != nil {
		return "", fmt.Errorf("failed to encode image: %w", err)
	}

	return base64.StdEncoding.EncodeToString(out.Bytes()), nil
}
