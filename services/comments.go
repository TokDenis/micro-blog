package services

import (
	"encoding/json"
	"github.com/TokDenis/micro-blog/types"
	"io"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Comments struct {
	commentsChan chan types.Comment
	buffer       map[int][]*types.Comment // [post_id]
	bufferM      sync.RWMutex
}

const CommentsPath = "db/comments/"

func NewCommentsService() *Comments {
	os.MkdirAll(CommentsPath, os.ModePerm)

	c := Comments{
		commentsChan: make(chan types.Comment, 100),
		buffer:       make(map[int][]*types.Comment),
	}

	go c.serv()

	return &c
}

func (c *Comments) Consume(postId int, msg types.Comment) {
	c.bufferM.Lock()
	c.buffer[postId] = append(c.buffer[postId], &msg)
	c.bufferM.Unlock()
}

func (c *Comments) GetComments(postId int) ([]*types.Comment, error) {
	f, err := os.OpenFile(CommentsPath+"/"+strconv.Itoa(postId), os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var comments []*types.Comment

	err = json.Unmarshal(b, &comments)
	if err != nil {
		return nil, err
	}

	res := comments[:0]

	for _, comment := range comments {
		if comment.IsDeleted {
			continue
		}
		res = append(res, comment)
	}

	return res, err
}

func (c *Comments) AppendNewComments(postId int, newComments []*types.Comment) error {
	if len(newComments) == 0 {
		return nil
	}

	f, err := os.OpenFile(CommentsPath+"/"+strconv.Itoa(postId), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	if len(b) != 0 {
		var oldComments []*types.Comment

		err = json.Unmarshal(b, &oldComments)
		if err != nil {
			return err
		}

		newComments = append(newComments, oldComments...)
	}

	sort.Slice(newComments, func(i, j int) bool { return newComments[i].Created.Before(newComments[j].Created) })

	for i, comment := range newComments {
		comment.Id = i
	}

	b, err = json.Marshal(&newComments)
	if err != nil {
		return err
	}

	err = f.Truncate(int64(len(b)))
	if err != nil {
		return err
	}

	_, err = f.WriteAt(b, 0)
	if err != nil {
		return err
	}

	return err
}

func (c *Comments) serv() {
	tic := time.NewTicker(time.Second)
	for {
		<-tic.C
		c.bufferM.Lock()
		for postId, comments := range c.buffer {
			err := c.AppendNewComments(postId, comments)
			if err != nil {

				continue
			}
			delete(c.buffer, postId)
		}
		c.bufferM.Unlock()
	}

}
