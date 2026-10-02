package main

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Valida o ID token do Cognito (RS256, iss, aud, exp) com as chaves publicas do pool.

type keyCache struct {
	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

var jwksCache = &keyCache{keys: map[string]*rsa.PublicKey{}}

func issuer() string {
	return fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", os.Getenv("AWS_REGION"), os.Getenv("USER_POOL_ID"))
}

func (c *keyCache) get(kid string) (*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if k, ok := c.keys[kid]; ok {
		return k, nil
	}
	// kid desconhecido: rebusca, no maximo 1x por minuto
	if time.Since(c.fetched) < time.Minute && len(c.keys) > 0 {
		return nil, errors.New("kid desconhecido")
	}
	resp, err := http.Get(issuer() + "/.well-known/jwks.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var doc struct {
		Keys []struct{ Kid, N, E string } `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	c.fetched = time.Now()
	for _, k := range doc.Keys {
		n, err1 := base64.RawURLEncoding.DecodeString(k.N)
		e, err2 := base64.RawURLEncoding.DecodeString(k.E)
		if err1 != nil || err2 != nil {
			continue
		}
		c.keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	if k, ok := c.keys[kid]; ok {
		return k, nil
	}
	return nil, errors.New("kid desconhecido")
}

type user struct{ Sub, Email string }

func verificarToken(raw string) (*user, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		return jwksCache.get(kid)
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(issuer()),
		jwt.WithAudience(os.Getenv("CLIENT_ID")), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	if claims["token_use"] != "id" {
		return nil, errors.New("use o id token")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, errors.New("sem sub")
	}
	email, _ := claims["email"].(string)
	return &user{Sub: sub, Email: email}, nil
}
