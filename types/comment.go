package types

import "time"

type Comment struct {
	Id        int       `json:"id"`
	UserName  string    `json:"user_name"`
	Content   string    `json:"content"`
	IsDeleted bool      `json:"is_deleted"`
	Created   time.Time `json:"created"`
}
