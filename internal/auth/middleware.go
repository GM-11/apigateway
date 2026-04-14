package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const claimsKey contextKey = "claims"

var Cache *KeyCache = NewKeyCache(time.Hour * 24 * 2)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")

		prefix := "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			log.Printf("Malformed authorization header: Invalid Authorization code")
			http.Error(w, "Malformed authorization header", http.StatusUnauthorized)
			return
		}

		reqToken := strings.TrimPrefix(authHeader, prefix)
		tokenParts := strings.Split(reqToken, ".")
		if len(tokenParts) != 3 {
			log.Printf("Invalid Token")
			http.Error(w, "Malformed token", http.StatusUnauthorized)
			return
		}
		var header struct {
			Kid string `json:"kid"`
			Alg string `json:"alg"`
		}

		if header.Alg != "RS256" {
			log.Printf("Unsupported algorithm: %s", header.Alg)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		headerBytes, err := base64.RawURLEncoding.DecodeString(tokenParts[0])
		if err != nil {
			log.Printf("Malformed token: invalid base64 in header")
			http.Error(w, "Malformed token: invalid base64 in header", http.StatusUnauthorized)
			return
		}
		if err = json.Unmarshal(headerBytes, &header); err != nil {
			log.Printf("Failed to parse token header: %v", err)
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		pubkey, err := Cache.GetKey(header.Kid)
		if err != nil {
			log.Printf("Failed to get public key for kid %s: %v", header.Kid, err.Error())
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		signingInput := tokenParts[0] + "." + tokenParts[1]

		signingHash := sha256.Sum256([]byte(signingInput))

		signinBytes, err := base64.RawURLEncoding.DecodeString(tokenParts[2])
		if err != nil {
			log.Printf("Malformed token: invalid base64 in signature")
			http.Error(w, "Malformed token", http.StatusUnauthorized)
			return
		}

		err = rsa.VerifyPKCS1v15(pubkey, crypto.SHA256, signingHash[:], signinBytes)

		if err != nil {

			log.Printf("Token signature verification failed: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var claims struct {
			Sub string `json:"sub"`
			Exp int64  `json:"exp"`
			Iat int64  `json:"iat"`
		}

		payloadBytes, err := base64.RawURLEncoding.DecodeString(tokenParts[1])
		if err != nil {
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}

		if err := json.Unmarshal(payloadBytes, &claims); err != nil {
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}

		if time.Now().Unix() > claims.Exp {
			http.Error(w, "Token Expired", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
