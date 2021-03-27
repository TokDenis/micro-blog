package main

import (
	"github.com/TokDenis/micro-blog/services"
	"github.com/rs/zerolog/log"
	"os"
)

var api *services.Api

func main() {
	os.Mkdir("db", os.ModePerm)
	var err error

	api, err = services.NewApi()
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	select {}
}
