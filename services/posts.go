package services

import (
	"encoding/json"
	"github.com/TokDenis/micro-blog/types"
	"github.com/karrick/godirwalk"
	"github.com/rs/zerolog/log"
	"os"
	"sort"
	"strconv"
	"time"
)

type Post struct {
	postIds      []int
	validPostIds []int
	timeIndex    *PostIndex
	stats        *Stats
}

func NewPost(stats *Stats) (*Post, error) {
	dbPath := "db/posts/"
	os.MkdirAll(dbPath, os.ModePerm)
	var filesCounter int

	err := godirwalk.Walk(dbPath, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if de.IsDir() {
				return nil
			}
			filesCounter++
			return nil
		},
		Unsorted: true,
	})
	if err != nil {
		return nil, err
	}

	ind, err := NewPostIndex()
	if err != nil {
		return nil, err
	}

	if filesCounter != 0 {
		filesCounter -= 1
	} else {
		filesCounter = -1
	}

	log.Info().Msgf("last post %d", filesCounter)

	p := &Post{
		stats:     stats,
		timeIndex: ind,
	}

	var postIds []int

	for i := 0; i <= filesCounter; i++ {
		postIds = append(postIds, i)
		post, err := p.ReadPost(i)
		if err != nil {
			return nil, err
		}

		if post.IsValid() {
			p.addValidPost(i)
		}
	}

	p.postIds = postIds

	return p, err
}

func (p *Post) CreatePost(req types.NewPostReq, user *types.UserInfo) (id int, err error) {
	id = p.postIds[len(p.postIds)-1] + 1

	f, err := os.OpenFile("db/posts/"+strconv.Itoa(id), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return -1, err
	}

	defer f.Close()

	post := types.Post{
		Id:        id,
		Name:      req.Name,
		ShortPost: req.ShortPost,
		MainPost:  req.MainPost,
		PostedBy:  user.Name,
		Created:   time.Now(),
	}

	b, err := json.Marshal(&post)
	if err != nil {
		return -1, err
	}

	_, err = f.Write(b)
	if err != nil {
		return -1, err
	}

	p.postIds = append(p.postIds, id)

	err = p.timeIndex.Append(post.Id, post.Created)
	if err != nil {
		return -1, err
	}

	err = p.stats.CreateStats(id)
	if err != nil {
		return -1, err
	}

	return id, nil
}

func (p *Post) DayTop(ts time.Time) (posts []*types.Post, err error) {
	posts, err = p.PostsByDay(ts)
	if err != nil {
		return nil, err
	}

	var postsIds []int
	for _, post := range posts {
		postsIds = append(postsIds, post.Id)
	}

	stats, err := p.stats.TopReadPostsStats(postsIds)
	if err != nil {
		return nil, err
	}

	for _, post := range posts {
		post.Stats = stats[post.Id]
	}

	sort.Slice(posts, func(i, j int) bool { return posts[i].Stats.Views > posts[j].Stats.Views })

	return posts, err
}

func (p *Post) GetPosts(from, count int) (posts []*types.Post, err error) {
	for i := from; i <= from+count; i++ {
		post, err := p.ReadPost(i)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, err
}

// LastPosts last 5 posts
func (p *Post) LastPosts(page int) (posts []*types.Post, err error) {
	fromPostId := len(p.validPostIds) - 1 - page*5
	if fromPostId < 0 {
		return nil, nil
	}

	for i := fromPostId; i >= 0; i-- {
		post, err := p.ReadPost(p.validPostIds[i])
		if err != nil {
			return nil, err
		}
		if post == nil {
			continue
		}

		posts = append(posts, post)
		if len(posts) == 5 {
			break
		}
	}
	return posts, err
}

func (p *Post) PostsByDay(ts time.Time) (posts []*types.Post, err error) {
	postsIds, err := p.timeIndex.PostsByDay(ts)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(postsIds); i++ {
		post, err := p.ReadPost(int(postsIds[i]))
		if err != nil {
			return nil, err
		}

		if !post.IsValid() {
			continue
		}

		posts = append(posts, post)
	}
	return posts, err
}

func (p *Post) ReadPost(id int) (*types.Post, error) {
	if id < 0 {
		return nil, nil
	}
	b, err := os.ReadFile("db/posts/" + strconv.Itoa(id))
	if err != nil {
		return nil, err
	}

	var post types.Post

	err = json.Unmarshal(b, &post)
	if err != nil {
		return nil, err
	}

	p.stats.CountView(post.Id)

	return &post, err
}

func (p *Post) PostsPages() int {
	return len(p.validPostIds)/5 + 1
}

func (p *Post) addValidPost(id int) {
	p.validPostIds = append(p.validPostIds, id)
	sort.Slice(p.validPostIds, func(i, j int) bool { return p.validPostIds[i] < p.validPostIds[j] })
}

func (p *Post) unValidPost(id int) {
	filter := p.validPostIds[:0]

	for i := 0; i < len(p.validPostIds); i++ {
		if p.validPostIds[i] == id {
			continue
		}
		filter = append(filter, p.validPostIds[i])
	}

	p.validPostIds = filter

	sort.Slice(p.validPostIds, func(i, j int) bool { return p.validPostIds[i] < p.validPostIds[j] })
}

func (p *Post) Validate(id int, validity bool) error {
	post, err := p.ReadPost(id)
	if err != nil {
		return err
	}

	post.IsApproved = validity

	err = p.setPost(id, post)
	if err != nil {
		return err
	}

	if validity {
		p.addValidPost(id)
	} else {
		p.unValidPost(id)
	}

	return err
}

func (p *Post) setPost(id int, post *types.Post) error {
	f, err := os.OpenFile("db/posts/"+strconv.Itoa(id), os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}

	defer f.Close()

	b, err := json.Marshal(post)
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
