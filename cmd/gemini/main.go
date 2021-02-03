package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"os"

	"gopkg.in/gemini"
)

// TODO: gemini.conman.org/test/torture/
// - 0013

var identityCertFile = flag.String("identity-cert", "", "identity cert file to use for requests")
var identityKeyFile = flag.String("identity-key", "", "identity key file to use for requests")

func main() {
	flag.Parse()

	client := gemini.Client{}

	if *identityCertFile != "" && *identityKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(*identityCertFile, *identityKeyFile)
		if err != nil {
			panic(err.Error())
		}
		client.Identity = &cert
	}

	for _, addr := range flag.Args() {
		req, err := gemini.NewRequest(addr)
		if err != nil {
			panic(err.Error())
		}

		resp, err := client.Do(req)
		if err != nil {
			panic(err.Error())
		}
		defer resp.Body.Close()

		fmt.Println(resp.Status, resp.Meta)

		if !resp.IsSuccess() {
			return
		}

		fmt.Println()

		mime, params, err := resp.MediaType()
		fmt.Println("MediaType:", mime)
		fmt.Printf("Params: %+v\n", params)

		fmt.Println()

		_, err = io.Copy(os.Stdout, resp.Body)
		if err != nil {
			panic(err.Error())
		}
	}
}
