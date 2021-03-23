package types

type User struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type UserInfo struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}
