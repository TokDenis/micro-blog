package services

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"github.com/TokDenis/micro-blog/types"
	"os"
)

type Auth struct {
}

func NewAuth() *Auth {
	os.MkdirAll("db/users/", os.ModePerm)
	return &Auth{}
}

// Signup - registration in blog
func (a *Auth) Signup(req types.NewUserReq) error {
	err := a.checkInfoCorrection(&req)
	if err != nil {
		return err
	}

	f, err := os.OpenFile("db/users/"+req.Email, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}

	defer f.Close()

	b, err := json.Marshal(types.User{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})

	_, err = f.Write(b)
	if err != nil {
		return err
	}

	return nil
}

func (a *Auth) checkInfoCorrection(req *types.NewUserReq) error {
	// check that password was hashed
	if len(req.Password) != sha256.BlockSize {
		return ErrIncorrectPassword
	}
	// check is user already exist
	if _, err := os.Open("db/users/" + req.Email); err == nil {
		return ErrUserExist
	}

	return nil
}

func (a *Auth) Sigin(email, password string) (*types.User, error) {
	b, err := os.ReadFile("db/users/" + email)
	if err != nil {
		return nil, err
	}

	var user types.User
	err = json.Unmarshal(b, &user)
	if err != nil {
		return nil, err
	}

	if user.Password != password {
		return nil, ErrUnauthorized
	}

	return &user, nil
}

func (a *Auth) UserInfo(email string) (*types.UserInfo, error) {
	b, err := os.ReadFile("db/users/" + email)
	if err != nil {
		return nil, err
	}

	var user types.UserInfo

	err = json.Unmarshal(b, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

var ErrIncorrectPassword = errors.New("incorrect password")
var ErrIncorrectEmail = errors.New("incorrect email")
var ErrUserExist = errors.New("exist")
var ErrUnauthorized = errors.New("unauthorized")
