package main

import (
	"log"
	"shadectl"
	"shadectl/adapters/http"
	"shadectl/adapters/somfy"
)

func main() {
	motorAddress := [3]byte{0x0c, 0x85, 0xae}
	motor, err := somfy.New(motorAddress)
	if err != nil {
		log.Fatalln("unable to create motor adapter", err)
	}
	svc := shadectl.NewService(motor)
	http.NewHTTPPrimaryAdapter(svc)
}