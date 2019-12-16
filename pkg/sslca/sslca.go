// Managing a CA and signing server certs.
// Used in a setting where we control both the servers and clients.
// Some code borrowed from https://golang.org/src/crypto/tls/generate_cert.go
package sslca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"github.com/function61/gokit/cryptoutil"
	"log"
	"math/big"
	"time"
)

func SelfSignedServerCert(hostname string, organisationName string, privateKeyPem []byte) ([]byte, error) {
	privateKey, err := cryptoutil.ParsePemEncodedPrivateKey(privateKeyPem)
	if err != nil {
		return nil, err
	}

	publicKey, err := cryptoutil.PublicKeyFromPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now().Add(time.Hour * -1) // account for clock drift
	notAfter := notBefore.AddDate(20, 0, 0)     // years

	certTemplate := &x509.Certificate{
		SerialNumber: generateSerialNumber(),
		Subject: pkix.Name{
			Organization: []string{organisationName},
		},

		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},

		DNSNames: []string{hostname},
	}

	certDer, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, publicKey, privateKey)
	if err != nil {
		return nil, err
	}

	return cryptoutil.MarshalPemBytes(certDer, cryptoutil.PemTypeCertificate), nil
}

func GenEcP256PrivateKeyPem() ([]byte, error) {
	// why EC: https://blog.cloudflare.com/ecdsa-the-digital-signature-algorithm-of-a-better-internet/
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	privateKeyX509, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	return cryptoutil.MarshalPemBytes(privateKeyX509, cryptoutil.PemTypeEcPrivateKey), nil
}

func generateSerialNumber() *big.Int {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	return serialNumber
}
