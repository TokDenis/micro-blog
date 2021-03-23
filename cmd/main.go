package main

import (
	"github.com/TokDenis/micro-blog/services"
	"github.com/rs/zerolog/log"
)

var api *services.Api

func main() {
	var err error

	api, err = services.NewApi()
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	select {}
}
