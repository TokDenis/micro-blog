package services

import (
	"crypto/rand"
	"encoding/hex"
	"os"
)

type Tokens struct {
}

func NewTokens() *Tokens {
	return &Tokens{}
}

func (t *Tokens) MakeToken(email string) (token string, err error) {
	b := make([]byte, 20)

	_, err = rand.Read(b)
	if err != nil {
		return "", err
	}

	token = hex.EncodeToString(b)

	f, err := os.Create("db/tokens/" + token)
	if err != nil {
		return "", err
	}

	defer f.Close()

	_, err = f.Write([]byte(email))
	if err != nil {
		return "", err
	}

	return token, err
}

func (t *Tokens) EmailFromToken(token string) (email string, err error) {
	b, err := os.ReadFile("db/tokens/" + token)
	if err != nil {
		return "", err
	}

	return string(b), err
}
