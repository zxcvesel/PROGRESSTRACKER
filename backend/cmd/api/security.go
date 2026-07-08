package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

func randomToken(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func tokenHashFromRequest(r *http.Request) string {
	token, ok := bearerToken(r)
	if !ok {
		return ""
	}
	return tokenHash(token)
}

func isStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasUpper := false
	hasDigit := false
	hasSpecial := false
	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case (char >= 'a' && char <= 'z') || char == ' ':
			continue
		default:
			hasSpecial = true
		}
	}

	return hasUpper && hasDigit && hasSpecial
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	key := argon2.IDKey([]byte(password), salt, passwordTime, passwordMemory, passwordThreads, passwordKeyBytes)
	return fmt.Sprintf(
		"%s$v=19$m=%d,t=%d,p=%d$%s$%s",
		passwordHashName,
		passwordMemory,
		passwordTime,
		passwordThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func verifyPassword(password string, storedHash string) bool {
	parts := strings.Split(storedHash, "$")
	if len(parts) == 4 && parts[0] == legacyHashName {
		return verifyLegacyPassword(password, parts)
	}

	if len(parts) != 5 || parts[0] != passwordHashName || parts[1] != "v=19" {
		return false
	}

	var memory uint32
	var timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads); err != nil {
		return false
	}
	if memory == 0 || timeCost == 0 || threads == 0 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	key := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(len(expected)))
	return subtle.ConstantTimeCompare(key, expected) == 1
}

func passwordNeedsRehash(storedHash string) bool {
	return !strings.HasPrefix(storedHash, passwordHashName+"$")
}

func verifyLegacyPassword(password string, parts []string) bool {
	rounds, err := strconv.Atoi(parts[1])
	if err != nil || rounds <= 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}

	key := deriveLegacyPasswordKey([]byte(password), salt, rounds, len(expected))
	return subtle.ConstantTimeCompare(key, expected) == 1
}

func deriveLegacyPasswordKey(password []byte, salt []byte, rounds int, keyLength int) []byte {
	hashLength := sha256.Size
	blockCount := int(math.Ceil(float64(keyLength) / float64(hashLength)))
	derived := make([]byte, 0, blockCount*hashLength)

	for block := 1; block <= blockCount; block++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		sum := mac.Sum(nil)
		blockBytes := append([]byte(nil), sum...)

		for round := 1; round < rounds; round++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(sum)
			sum = mac.Sum(nil)
			for index := range blockBytes {
				blockBytes[index] ^= sum[index]
			}
		}

		derived = append(derived, blockBytes...)
	}

	return derived[:keyLength]
}
