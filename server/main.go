package main

import (
	"github.com/emersion/go-smtp"
	"log"
	"github.com/moisespsena/go-remote-smtp-sender-proxy"
)

func main() {
	config, err := rsmtp.LoadConfig("server/config.yaml")
	if err != nil {
		panic(err)
	}
	be := rsmtp.NewBackend(config)
	defer be.Done()
	err = be.Start()
	if err != nil {
		panic(err)
	}

	s := smtp.NewServer(be)

	s.Addr = ":1025"
	s.Domain = "localhost"
	s.MaxIdleSeconds = 300
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true

	log.Println("Starting server at", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}