package main

import (
	"context"
	"flag"
	"os"

	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/mailer/pkg/client"
	"github.com/ViBiOh/mailer/pkg/model"
)

func main() {
	fs := flag.NewFlagSet("client", flag.ExitOnError)

	loggerConfig := logger.Flags(fs, "logger")
	mailerConfig := client.Flags(fs, "mailer")
	recipient := flags.New("", "").Name("Recipient").Default("").Label("Recipient").ToString(fs)

	logger.Fatal(fs.Parse(os.Args[1:]))

	logger.Global(logger.New(loggerConfig))
	defer logger.Close()

	client, err := client.New(mailerConfig)
	logger.Fatal(err)
	defer client.Close()

	logger.Fatal(client.Send(context.Background(), *model.NewMailRequest().From("mailer@vibioh.fr").As("Client").To(*recipient).Template("hello")))
}
