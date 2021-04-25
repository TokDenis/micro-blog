package services

import (
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"strings"
	"time"
)

func StartProxy() {
	fs := &fasthttp.FS{
		Root:       "/www/micro-blog",
		IndexNames: []string{"index.html"},
		Compress:   true,
	}

	pages := fs.NewRequestHandler()

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		writeCors(ctx)
		if string(ctx.Method()) == fasthttp.MethodOptions {
			return
		}

		switch {
		case strings.HasPrefix(string(ctx.Path()), "/api"):
			err := proxy("localhost:8080", ctx)
			if err != nil {
				log.Error().Err(err).Send()
				return
			}
		default:
			pages(ctx)
		}

	}

	s := &fasthttp.Server{
		Handler: requestHandler,
		Name:    "micro-blog-proxy",
	}

	log.Info().Msg("micro-blog-proxy ok")

	if err := s.ListenAndServe(":80"); err != nil {
		log.Fatal().Err(err).Send()
	}
}

var (
	corsAllowHeaders     = "*"
	corsAllowMethods     = "GET,OPTIONS"
	corsAllowCredentials = "true"
)

func proxy(server string, ctx *fasthttp.RequestCtx) error {
	ctx.Request.SetHost(server)

	err := fasthttp.DoTimeout(&ctx.Request, &ctx.Response, time.Second*10)
	if err != nil {
		return err
	}

	return err
}

func writeCors(ctx *fasthttp.RequestCtx) {
	org := string(ctx.Request.Header.Peek("Origin"))
	if len(org) == 0 {
		return
	}

	switch {
	case strings.Contains(org, "localhost"):
	case strings.Contains(org, "maki-station.com"):
	default:
		return
	}

	ctx.Response.Header.Set("Access-Control-Allow-Credentials", corsAllowCredentials)
	ctx.Response.Header.Set("Access-Control-Allow-Headers", corsAllowHeaders)
	ctx.Response.Header.Set("Access-Control-Allow-Methods", corsAllowMethods)
	ctx.Response.Header.Set("Access-Control-Allow-Origin", org)
}
