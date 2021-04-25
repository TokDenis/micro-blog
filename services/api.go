package services

import (
	"encoding/json"
	"errors"
	"github.com/TokDenis/micro-blog/types"
	"github.com/lab259/cors"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttprouter"
	"io/fs"
	"strconv"
	"time"

	"github.com/kataras/go-sessions/v3"
)

type Api struct {
	auth     *Auth
	post     *Post
	token    *Tokens
	stats    *Stats
	comments *Comments
}

const (
	TokenKey = "x-token"
)

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
		auth:     NewAuth(),
		post:     post,
		token:    NewTokens(),
		stats:    stats,
		comments: NewCommentsService(),
	}
	r.POST("/api/v1/adm/valid", api.AuthMiddleware(api.ValidatePost))

	r.GET("/api/v1/comments", api.Comments)
	r.POST("/api/v1/comments/new", api.AuthMiddleware(api.NewComment))

	r.POST("/api/v1/auth/newuser", api.NewUser)
	r.POST("/api/v1/auth/login", api.LoginUser)
	r.POST("/api/v1/auth/logout", api.LoginUser)
	r.GET("/api/v1/auth/userinfo", api.AuthMiddleware(api.UserInfo))

	r.GET("/api/v1/post/pages", api.PostsPages)
	r.GET("/api/v1/post/daytop", api.DayTopPosts)
	//r.GET("/api/v1/post/next", api.GetPosts)
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
		a.internalErr(ctx, err)
		return
	}

	token, err := a.token.MakeToken(newUserReq.Email)
	if err != nil {
		a.internalErr(ctx, err)
		_, _ = ctx.Write([]byte(err.Error()))
		return
	}

	ses := sessions.StartFasthttp(ctx)
	ses.Set(TokenKey, token)

	ctx.SetStatusCode(fasthttp.StatusOK)
	//_, _ = ctx.Write([]byte(token))
}

func (a *Api) internalErr(ctx *fasthttp.RequestCtx, err error) {
	log.Error().Err(err).Send()
	ctx.SetStatusCode(fasthttp.StatusInternalServerError)
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
		if errors.Is(err, fs.ErrNotExist) {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}
		a.internalErr(ctx, err)
		return
	}

	token, err := a.token.MakeToken(user.Email)
	if err != nil {
		a.internalErr(ctx, err)
		return
	}

	ses := sessions.StartFasthttp(ctx)
	ses.Set(TokenKey, token)

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (a *Api) UserInfo(ctx *fasthttp.RequestCtx, _ fasthttprouter.Params) {
	email := ctx.UserValue("_email").(string)

	user, err := a.auth.UserInfo(email)
	if err != nil {
		a.internalErr(ctx, err)
		return
	}

	b, err := json.Marshal(&user)
	if err != nil {
		a.internalErr(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(b)
}

func (a *Api) AuthMiddleware(next fasthttprouter.Handle) fasthttprouter.Handle {
	return func(ctx *fasthttp.RequestCtx, p fasthttprouter.Params) {
		ses := sessions.StartFasthttp(ctx)
		token, ok := ses.Get(TokenKey).(string)
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
	var page int
	var err error

	pageString := string(ctx.QueryArgs().Peek("page"))
	if pageString != "" {
		page, err = strconv.Atoi(pageString)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}
		if page > 0 {
			page--
		}
	}

	posts, err := a.post.LastPosts(page)
	if err != nil {
		a.internalErr(ctx, err)
		return
	}

	b, err := json.Marshal(&posts)
	if err != nil {
		a.internalErr(ctx, err)
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
		a.internalErr(ctx, err)
		return
	}

	b, err := json.Marshal(&posts)
	if err != nil {
		a.internalErr(ctx, err)
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
		a.internalErr(ctx, err)
		return
	}

	b, err := json.Marshal(&posts)
	if err != nil {
		a.internalErr(ctx, err)
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
		a.internalErr(ctx, err)
		return
	}

	if !post.IsValid() {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	b, err := json.Marshal(&post)
	if err != nil {
		a.internalErr(ctx, err)
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
		a.internalErr(ctx, err)
		return
	}

	id, err := a.post.CreatePost(postReq, userInfo)
	if err != nil {
		a.internalErr(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write([]byte(strconv.Itoa(id)))
}

func (a *Api) ReadStats(ctx *fasthttp.RequestCtx, p fasthttprouter.Params) {
	var ids []int

	err := json.Unmarshal(ctx.QueryArgs().Peek("ids"), &ids)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	var stats []*types.Stats

	for _, id := range ids {
		stat, err := a.stats.ReadStats(id)
		if err != nil {
			a.internalErr(ctx, err)
			return
		}
		stats = append(stats, stat)
	}

	b, err := json.Marshal(&stats)
	if err != nil {
		a.internalErr(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(b)
}

func (a *Api) PostsPages(ctx *fasthttp.RequestCtx, p fasthttprouter.Params) {
	count := a.post.PostsPages()

	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write([]byte(strconv.Itoa(count)))
}

func (a *Api) NewComment(ctx *fasthttp.RequestCtx, _ fasthttprouter.Params) {
	var commentReq types.NewCommentReq

	err := json.Unmarshal(ctx.PostBody(), &commentReq)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	email := ctx.UserValue("_email").(string)

	a.comments.Consume(commentReq.PostId, types.Comment{
		Id:        0,
		UserName:  email,
		Content:   commentReq.Content,
		IsDeleted: false,
		Created:   time.Now(),
	})

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (a *Api) Comments(ctx *fasthttp.RequestCtx, _ fasthttprouter.Params) {
	id, err := strconv.Atoi(string(ctx.QueryArgs().Peek("id")))
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	comments, err := a.comments.GetComments(id)
	if err != nil {
		a.internalErr(ctx, err)
		return
	}

	b, err := json.Marshal(comments)
	if err != nil {
		a.internalErr(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	_, _ = ctx.Write(b)
}

func (a *Api) LogoutUser(ctx *fasthttp.RequestCtx, _ fasthttprouter.Params) {
	ses := sessions.StartFasthttp(ctx)
	token, ok := ses.Get(TokenKey).(string)
	if !ok {
		a.internalErr(ctx, errors.New("can not get TokenKey"))
		return
	}

	err := a.token.DeleteToken(token)
	if err != nil {
		a.internalErr(ctx, err)
		return
	}

	ok = ses.Delete(TokenKey)
	if !ok {
		a.internalErr(ctx, errors.New("can not delete TokenKey"))
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (a *Api) ValidatePost(ctx *fasthttp.RequestCtx, _ fasthttprouter.Params) {
	if string(ctx.PostBody()) != types.AdminWord {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(string(ctx.QueryArgs().Peek("id")))
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	err = a.post.Validate(id, ctx.QueryArgs().GetBool("v"))
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}
