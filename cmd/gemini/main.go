package main

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/gemini"
)

func main() {
	resp, err := gemini.Get(os.Args[1])
	if err != nil {
		panic(err.Error())
	}
	defer resp.Body.Close()

	fmt.Println(resp.Status, resp.Meta)

	if !resp.IsSuccess() {
		return
	}

	fmt.Println()

	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		panic(err.Error())
	}
}
