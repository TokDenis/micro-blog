package types

import "time"

type Post struct {
	Id         int       `json:"id"`
	Name       string    `json:"name"`
	ShortPost  string    `json:"short_post"`
	MainPost   string    `json:"main_post"`
	PostedBy   string    `json:"posted_by"`
	Created    time.Time `json:"created"`
	Stats      *Stats    `json:"stats,omitempty"`
	IsApproved bool      `json:"is_approved"`
}

func (p *Post) IsValid() bool {
	return p.IsApproved
}

type Stats struct {
	Id    int   `json:"id"`
	Views int64 `json:"views"`
}
