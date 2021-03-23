package types

type NewPostReq struct {
	Name      string `json:"name"`
	ShortPost string `json:"short_post"`
	MainPost  string `json:"main_post"`
}

type NewUserReq struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
