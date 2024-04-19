package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"math/big"
	"net/http"
	"time"
)

var (
	addrHTTP  = flag.String("http", ":30081", "Address to listen on for HTTP")
	addrHTTPS = flag.String("https", ":30082", "Address to listen on for HTTPS")
	path      = flag.String("path", ".", "Path to serve")
)

func main() {
	flag.Parse()

	http.Handle("/", http.FileServer(http.Dir(*path)))

	// HTTP server.
	go http.ListenAndServe(*addrHTTP, nil)

	// HTTPS server.
	cert := GenCert()
	server := &http.Server{
		Addr:      *addrHTTPS,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{*cert}},
	}
	server.ListenAndServeTLS("", "")
}

func GenCert() *tls.Certificate {
	// Build the certificate template.
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{Organization: []string{"Test Cert Org"}},

		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(1 * time.Hour),

		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},

		BasicConstraintsValid: true,

		DNSNames: []string{"localhost"},
	}

	// Generate a private key.
	privK, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}

	derBytes, err := x509.CreateCertificate(
		rand.Reader, &tmpl, &tmpl, privK.Public(), privK)
	if err != nil {
		panic(err)
	}

	return &tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  privK,
	}
}
