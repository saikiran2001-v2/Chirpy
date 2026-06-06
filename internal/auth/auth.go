package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	hashedPass, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", errors.New("cannot create hash from the given password")
	}

	return hashedPass, nil
}

func CheckPaswordHash(password, Hash string) (bool, error) {
	isMatch, err := argon2id.ComparePasswordAndHash(password, Hash)
	if err != nil {
		return false, errors.New("something went wrong")
	}

	return isMatch, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{Issuer: "chirpy-access", IssuedAt: jwt.NewNumericDate(time.Now().UTC()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)), Subject: userID.String()})
	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", errors.New("cannot sign the token")
	}

	return tokenString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		_, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return uuid.UUID{}, errors.New("token invalid or expired")
	}

	userID, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.UUID{}, errors.New("something went wrong")
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		return uuid.UUID{}, errors.New("cannot parse user id")
	}

	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("cannot find auth header")
	}

	authSplit := strings.Split(authHeader, " ")
	if len(authSplit) != 2 || authSplit[0] != "Bearer" {
		return "", errors.New("malformed authorization header")
	}
	finalAuth := strings.TrimSpace(authSplit[1])
	return finalAuth, nil
}

func MakeRefreshToken() string {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return ""
	}
	rToknStr := hex.EncodeToString(key)
	return rToknStr
}
