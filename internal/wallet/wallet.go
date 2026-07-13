package wallet

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
)

func GenerateKeyPair() (string, string, error) {
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return "", "", err
	}
	return hex.EncodeToString(privKey), hex.EncodeToString(pubKey), nil
}

func GetPublicKey(privKeyHex string) (string, error) {
	privBytes, err := hex.DecodeString(privKeyHex)
	if err != nil || len(privBytes) != ed25519.PrivateKeySize {
		return "", errors.New("invalid private key")
	}
	privKey := ed25519.PrivateKey(privBytes)
	pubKey := privKey.Public().(ed25519.PublicKey)
	return hex.EncodeToString(pubKey), nil
}

func Sign(privKeyHex string, message string) (string, error) {
	privBytes, err := hex.DecodeString(privKeyHex)
	if err != nil || len(privBytes) != ed25519.PrivateKeySize {
		return "", errors.New("invalid private key format")
	}
	privKey := ed25519.PrivateKey(privBytes)
	sig := ed25519.Sign(privKey, []byte(message))
	return hex.EncodeToString(sig), nil
}

func Verify(pubKeyHex string, message string, sigHex string) bool {
	pubBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return false
	}
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}
	pubKey := ed25519.PublicKey(pubBytes)
	return ed25519.Verify(pubKey, []byte(message), sigBytes)
}
