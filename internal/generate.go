package internal

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"strings"
)

const DefaultKeyBits = 2048
const (
	KeyTypeRSA       = "rsa"
	KeyTypeECDSAP256 = "ecdsa_p256"
)

func GeneratePrivateKey() (*rsa.PrivateKey, error) {
	return GeneratePrivateKeyWithBits(DefaultKeyBits)
}

func NormalizeKeyBits(bits int) int {
	switch bits {
	case 2048, 3072, 4096:
		return bits
	default:
		return DefaultKeyBits
	}
}

func NormalizeKeyType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case KeyTypeECDSAP256, "ecdsa":
		return KeyTypeECDSAP256
	default:
		return KeyTypeRSA
	}
}

func GeneratePrivateKeyWithBits(bits int) (*rsa.PrivateKey, error) {
	bits = NormalizeKeyBits(bits)
	return rsa.GenerateKey(rand.Reader, bits)
}

func GenerateKeyPair(keyType string, keyBits int) (crypto.PrivateKey, crypto.PublicKey, error) {
	switch NormalizeKeyType(keyType) {
	case KeyTypeECDSAP256:
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		return privateKey, &privateKey.PublicKey, nil
	default:
		privateKey, err := GeneratePrivateKeyWithBits(keyBits)
		if err != nil {
			return nil, nil, err
		}
		return privateKey, &privateKey.PublicKey, nil
	}
}

func GenerateSerialNumber() *big.Int {
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	return serialNumber
}
