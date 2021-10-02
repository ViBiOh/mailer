package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/ViBiOh/mailer/pkg/client"
	"github.com/ViBiOh/mailer/pkg/model"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	fs := flag.NewFlagSet("client", flag.ExitOnError)

	mailerConfig := client.Flags(fs, "mailer")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	client, err := client.New(mailerConfig, prometheus.DefaultRegisterer)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	log.Printf("Mailer Client: %s\n", client)

	if err := client.Send(context.Background(), model.NewMailRequest().From("mailer@vibioh.fr").As("Client").To("customer@vibioh.fr").Template("hello")); err != nil {
		log.Fatal(err)
	}
}
