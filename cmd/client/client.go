package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/mailer/pkg/client"
	"github.com/ViBiOh/mailer/pkg/model"
)

func main() {
	fs := flag.NewFlagSet("client", flag.ExitOnError)
	fs.Usage = flags.Usage(fs)

	mailerConfig := client.Flags(fs, "mailer")

	_ = fs.Parse(os.Args[1:])

	client, err := client.New(mailerConfig, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	log.Printf("Mailer Client: %s\n", client)

	if err := client.Send(context.Background(), model.NewMailRequest().From("mailer@vibioh.fr").As("Client").To("customer@vibioh.fr").Template("hello")); err != nil {
		log.Fatal(err)
	}
}
