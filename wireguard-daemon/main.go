package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
)

var (
	//nolint
	tlsCertDir = "."
	//nolint
	tlsKeyDir  = "."
	wgPort     = 51820
	dataDir    = flag.String("data-dir", "", "Directory used for storage")
	listenAddr = flag.String("listen-address", ":8080", "Address to listen to")
	wgLinkName = flag.String("wg-device-name", "wg0", "WireGuard network device name")
	wgStatus   = false
)

func main() {
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()
	s := NewServer()

	startErr := s.Start()
	if startErr != nil {
		fmt.Print("Error starting server: ", startErr)
	}
	startAPIErr := s.StartAPI()
	if startAPIErr != nil {
		fmt.Println("Error starting API: ", startAPIErr)
	}
}

//nolint
func getTlsConfig() *tls.Config {
	caCertFile := filepath.Join(tlsCertDir, "ca.crt")
	certFile := filepath.Join(tlsCertDir, "server.crt")
	keyFile := filepath.Join(tlsKeyDir, "server.key")

	keyPair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}

	caCertPem, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		log.Fatal(err)
	}

	trustedCaPool := x509.NewCertPool()
	if !trustedCaPool.AppendCertsFromPEM(caCertPem) {
	}

	return &tls.Config{
		Certificates: []tls.Certificate{keyPair},
		MinVersion:   tls.VersionTLS12,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    trustedCaPool,
		CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384},
	}
}
