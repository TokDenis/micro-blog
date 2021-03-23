package services

import (
	"encoding/json"
	"errors"
	"github.com/TokDenis/micro-blog/types"
	"github.com/lab259/cors"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttprouter"
	"strconv"
	"time"

	"github.com/kataras/go-sessions/v3"
)

type Api struct {
	auth  *Auth
	post  *Post
	token *Tokens
	stats *Stats
}

func NewApi() (*Api, error) {
	r := fasthttprouter.New()

	cs := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:7777", "http://localhost:8080"},
		AllowedMethods: []string{
			fasthttp.MethodHead,
			fasthttp.MethodGet,
			fasthttp.MethodPost,
			fasthttp.MethodPut,
			fasthttp.MethodPatch,
			fasthttp.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	s := &fasthttp.Server{
		ReadTimeout:  time.Second * 5,
		IdleTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		Handler:      cs.Handler(r.Handler),
	}

	stats := NewStats()

	post, err := NewPost(stats)
	if err != nil {
		return nil, err
	}

	api := Api{
		auth:  NewAuth(),
		post:  post,
		token: NewTokens(),
		stats: stats,
	}

	r.POST("/api/v1/auth/newuser", api.NewUser)
	r.POST("/api/v1/auth/login", api.LoginUser)
	r.GET("/api/v1/auth/userinfo", api.AuthMiddleware(api.UserInfo))

	r.GET("/api/v1/post/daytop", api.DayTopPosts)
	r.GET("/api/v1/post/next", api.GetPosts)
	r.GET("/api/v1/post/last", api.LastPosts)
	r.GET("/api/v1/post", api.OpenPost)
	r.POST("/api/v1/post/new", api.AuthMiddleware(api.NewPost))

	r.GET("/api/v1/stats", api.ReadStats)

	go func() {
		err := s.ListenAndServe(":8080")
		if err != nil {
			log.Fatal().Err(err).Send()
		}
	}()

	return &api, err
}

func (a *Api) NewUser(ctx *fasthttp.RequestCtx, _ fasthttprouter.Params) {
	var newUserReq types.NewUserReq

	err := json.Unmarshal(ctx.PostBody(), &newUserReq)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	err = a.auth.Signup(newUserReq)
	if err != nil {
		if errors.Is(err, ErrUserExist) || errors.Is(err, ErrIncorrectEmail) || errors.Is(err, ErrIncorrectPassword) {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			_, _ = ctx.Write([]byte(err.Error()))
			return
		}
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	token, err := a.token.MakeToken(newUserReq.Email)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		_, _ = ctx.Write([]byte(err.Error()))
		return
	}

	ses := sessions.StartFasthttp(ctx)
	ses.Set("x-tokent", token)

	ctx.SetStatusCode(fasthttp.StatusOK)
	//_, _ = ctx.Write([]byte(token))
}

func (a *Api) LoginUser(ctx *fasthttp.RequestCtx, _ fasthttprouter.Params) {
	var userReq types.NewUserReq

	err := json.Unmarshal(ctx.PostBody(), &userReq)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	user, err := a.auth.Sigin(userReq.Email, userReq.Password)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	token, err := a.token.MakeToken(user.Email)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write([]byte(token))
}

func (a *Api) UserInfo(ctx *fasthttp.RequestCtx, _ fasthttprouter.Params) {
	email := ctx.UserValue("_email").(string)

	user, err := a.auth.UserInfo(email)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(&user)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(b)
}

func (a *Api) AuthMiddleware(next fasthttprouter.Handle) fasthttprouter.Handle {
	return func(ctx *fasthttp.RequestCtx, p fasthttprouter.Params) {
		ses := sessions.StartFasthttp(ctx)
		token, ok := ses.Get("x-tokent").(string)
		if !ok {
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			return
		}
		email, err := a.token.EmailFromToken(token)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			return
		}

		ctx.SetUserValue("_email", email)
		next(ctx, p)
	}
}

func (a *Api) LastPosts(ctx *fasthttp.RequestCtx, _ fasthttprouter.Params) {
	posts, err := a.post.LastPosts()
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(&posts)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(b)
}

func (a *Api) DayTopPosts(ctx *fasthttp.RequestCtx, _ fasthttprouter.Params) {
	ts, err := time.Parse("2006-01-02", string(ctx.QueryArgs().Peek("day")))
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	posts, err := a.post.DayTop(ts)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(&posts)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(b)
}

func (a *Api) GetPosts(ctx *fasthttp.RequestCtx, _ fasthttprouter.Params) {
	from, err := strconv.Atoi(string(ctx.QueryArgs().Peek("from")))
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	count, err := strconv.Atoi(string(ctx.QueryArgs().Peek("count")))
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	posts, err := a.post.GetPosts(from, count)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(&posts)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(b)
}

func (a *Api) OpenPost(ctx *fasthttp.RequestCtx, p fasthttprouter.Params) {
	id, err := strconv.Atoi(string(ctx.QueryArgs().Peek("id")))
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	post, err := a.post.ReadPost(id)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(&post)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(b)
}

func (a *Api) NewPost(ctx *fasthttp.RequestCtx, _ fasthttprouter.Params) {
	var postReq types.NewPostReq

	err := json.Unmarshal(ctx.PostBody(), &postReq)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	email := ctx.UserValue("_email").(string)

	userInfo, err := a.auth.UserInfo(email)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	id, err := a.post.CreatePost(postReq, userInfo)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write([]byte(strconv.Itoa(id)))
}

func (a *Api) ReadStats(ctx *fasthttp.RequestCtx, p fasthttprouter.Params) {
	id, err := strconv.Atoi(string(ctx.QueryArgs().Peek("id")))
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	stats, err := a.stats.ReadStats(id)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(&stats)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(b)
}
