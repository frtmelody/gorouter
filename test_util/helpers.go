package test_util

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/gorouter/config"
)

func SpecConfig(statusPort, proxyPort uint16, natsPorts ...uint16) *config.Config {
	return generateConfig(statusPort, proxyPort, natsPorts...)
}

func SpecSSLConfig(statusPort, proxyPort, SSLPort uint16, natsPorts ...uint16) *config.Config {
	c := generateConfig(statusPort, proxyPort, natsPorts...)

	c.EnableSSL = true

	key, cert := CreateKeyPair("potato.com")
	secondKey, secondCert := CreateKeyPair("potato2.com")

	c.TLSPEM = []string{
		fmt.Sprintf("%s\n%s", string(key), string(cert)),
		fmt.Sprintf("%s\n%s", string(secondKey), string(secondCert)),
	}
	c.SSLPort = SSLPort
	c.CipherString = "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"

	return c
}

func generateConfig(statusPort, proxyPort uint16, natsPorts ...uint16) *config.Config {
	c := config.DefaultConfig()

	c.Port = proxyPort
	c.Index = 2
	c.TraceKey = "my_trace_key"

	// Hardcode the IP to localhost to avoid leaving the machine while running tests
	c.Ip = "127.0.0.1"

	c.StartResponseDelayInterval = 1 * time.Second
	c.PublishStartMessageInterval = 10 * time.Second
	c.PruneStaleDropletsInterval = 0
	c.DropletStaleThreshold = 10 * time.Second
	c.PublishActiveAppsInterval = 0
	c.Zone = "z1"

	c.EndpointTimeout = 500 * time.Millisecond

	c.Status = config.StatusConfig{
		Port: statusPort,
		User: "user",
		Pass: "pass",
	}

	c.Nats = []config.NatsConfig{}
	for _, natsPort := range natsPorts {
		c.Nats = append(c.Nats, config.NatsConfig{
			Host: "localhost",
			Port: natsPort,
			User: "nats",
			Pass: "nats",
		})
	}

	c.Logging = config.LoggingConfig{
		Level:         "debug",
		MetronAddress: "localhost:3457",
		JobName:       "router_test_z1_0",
	}

	c.OAuth = config.OAuthConfig{
		TokenEndpoint:     "uaa.cf.service.internal",
		Port:              8443,
		SkipSSLValidation: true,
	}

	c.RouteServiceSecret = "kCvXxNMB0JO2vinxoru9Hg=="

	c.Tracing = config.Tracing{
		EnableZipkin: true,
	}

	return c
}

func CreateKeyPair(cname string) (keyPEM, certPEM []byte) {
	// generate a random serial number (a real cert authority would have some logic behind this)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	Expect(err).ToNot(HaveOccurred())

	subject := pkix.Name{Organization: []string{"xyz, Inc."}}
	if cname != "" {
		subject.CommonName = cname
	}

	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               subject,
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour), // valid for an hour
		BasicConstraintsValid: true,
	}

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	Expect(err).ToNot(HaveOccurred())
	certDER, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &privKey.PublicKey, privKey)
	Expect(err).ToNot(HaveOccurred())

	b := pem.Block{Type: "CERTIFICATE", Bytes: certDER}
	certPEM = pem.EncodeToMemory(&b)
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})

	return
}

func CreateECKeyPair(cname string) (keyPEM, certPEM []byte) {
	// generate a random serial number (a real cert authority would have some logic behind this)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	Expect(err).ToNot(HaveOccurred())

	subject := pkix.Name{Organization: []string{"xyz, Inc."}}
	if cname != "" {
		subject.CommonName = cname
	}

	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               subject,
		SignatureAlgorithm:    x509.ECDSAWithSHA256,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour), // valid for an hour
		BasicConstraintsValid: true,
	}

	elliptic := elliptic.P256()
	privKey, err := ecdsa.GenerateKey(elliptic, rand.Reader)
	Expect(err).ToNot(HaveOccurred())

	certDER, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &privKey.PublicKey, privKey)
	Expect(err).ToNot(HaveOccurred())

	b := pem.Block{Type: "CERTIFICATE", Bytes: certDER}
	certPEM = pem.EncodeToMemory(&b)
	privBytes, err := x509.MarshalECPrivateKey(privKey)
	Expect(err).ToNot(HaveOccurred())

	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privBytes,
	})
	// the values for oid came from https://golang.org/src/crypto/x509/x509.go?s=54495:54612#L290
	ecdsaOid, err := asn1.Marshal(asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2})
	paramPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PARAMETERS", Bytes: ecdsaOid})
	keyPEM = []byte(fmt.Sprintf("%s%s", paramPEM, keyPEM))
	return
}

func CreateCert(cname string) tls.Certificate {
	privKeyPEM, certPEM := CreateKeyPair(cname)
	tlsCert, err := tls.X509KeyPair(certPEM, privKeyPEM)
	Expect(err).ToNot(HaveOccurred())
	return tlsCert
}

func CreateECCert(cname string) tls.Certificate {
	privKeyPEM, certPEM := CreateECKeyPair(cname)
	tlsCert, err := tls.X509KeyPair(certPEM, privKeyPEM)
	Expect(err).ToNot(HaveOccurred())
	return tlsCert
}
