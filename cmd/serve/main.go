package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"flag"
	"fmt"
	"strings"

	"gopkg.in/gemini"
)

var identityCertFile = flag.String("identity-cert", "", "identity cert file to use for requests")
var identityKeyFile = flag.String("identity-key", "", "identity key file to use for requests")

func main() {
	flag.Parse()

	server := gemini.Server{
		TLS: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
			ClientAuth:         tls.RequestClientCert,
		},
		Handler: gemini.HandlerFunc(func(r *gemini.Request) (*gemini.Response, error) {
			fmt.Println("<--", strings.TrimSuffix(r.String(), "\r\n"))

			if len(r.TLS.PeerCertificates) != 0 {
				cert := r.TLS.PeerCertificates[0]

				hash := sha256.New()
				hash.Write(cert.Raw)
				fingerprint := hash.Sum(nil)

				var buf bytes.Buffer
				for i, f := range fingerprint {
					if i > 0 {
						fmt.Fprintf(&buf, ":")
					}
					fmt.Fprintf(&buf, "%02X", f)
				}
				fmt.Printf("Fingerprint: %s\n", buf.String())
			}

			return gemini.NewResponseString(gemini.StatusNotFound, "Not found", ""), nil
		}),
	}

	if *identityCertFile != "" && *identityKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(*identityCertFile, *identityKeyFile)
		if err != nil {
			panic(err.Error())
		}
		server.TLS.Certificates = []tls.Certificate{cert}
	}

	err := server.ListenAndServe()
	if err != nil {
		panic(err.Error())
	}

}
