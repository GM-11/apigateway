package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"os"
)

func FetchJWKS() (map[string]*rsa.PublicKey, error) {

	url := os.Getenv("JWKS_ENDPOINT")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
			Kty string `json:"kty"`
		} `json:"keys"`
	}

	json.NewDecoder(resp.Body).Decode(&response)

	pubkeys := make(map[string]*rsa.PublicKey)

	for _, key := range response.Keys {
		if key.Kty != "RSA" {
			continue
		}

		nDecode, err := base64.RawURLEncoding.DecodeString(key.N)

		if err != nil {
			log.Printf("Failed to decode n for key %s: %v", key.Kid, err)
			continue
		}

		eDecode, err := base64.RawURLEncoding.DecodeString(key.E)

		if err != nil {
			log.Printf("Failed to decode e for key %s: %v", key.Kid, err)
			continue
		}

		pubkey := rsa.PublicKey{
			N: new(big.Int).SetBytes(nDecode),
			E: int(new(big.Int).SetBytes(eDecode).Int64()),
		}

		pubkeys[key.Kid] = &pubkey
	}

	return pubkeys, nil
}
