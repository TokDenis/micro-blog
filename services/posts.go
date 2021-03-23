package services

import (
	"encoding/json"
	"github.com/TokDenis/micro-blog/types"
	"github.com/karrick/godirwalk"
	"github.com/rs/zerolog/log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Post struct {
	files     []int
	timeIndex *PostIndex
	stats     *Stats
}

func NewPost(stats *Stats) (*Post, error) {
	dbPath := "db/posts/"
	var files []int

	err := godirwalk.Walk(dbPath, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			file := strings.Replace(osPathname, dbPath, "", 1)
			if file != "" {
				fileId, err := strconv.Atoi(file)
				if err != nil {
					log.Info().Err(err).Send()
					return nil
				}
				files = append(files, fileId)
			}
			return nil
		},
		Unsorted: true,
	})
	if err != nil {
		return nil, err
	}

	sort.Ints(files)

	ind, err := NewPostIndex()
	if err != nil {
		return nil, err
	}

	return &Post{
		files:     files,
		stats:     stats,
		timeIndex: ind,
	}, err
}

func (p *Post) CreatePost(req types.NewPostReq, user *types.UserInfo) (id int, err error) {
	if len(p.files) != 0 {
		id = p.files[len(p.files)-1] + 1
	} else {
		id = 0
	}
	p.files = append(p.files, id)

	f, err := os.OpenFile("db/posts/"+strconv.Itoa(id), os.O_RDWR|os.O_CREATE, 0777)
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

	f.Write(b)

	err = p.timeIndex.Append(post.Id, post.Created)
	if err != nil {
		return 0, err
	}

	err = p.stats.CreateStats(id)
	if err != nil {
		return 0, err
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

// last 5 posts
func (p *Post) LastPosts() (posts []*types.Post, err error) {
	var postsIds []int

	if len(p.files) > 5 {
		postsIds = p.files[len(p.files)-5:]
	} else {
		postsIds = p.files
	}

	for i := 0; i < len(postsIds); i++ {
		post, err := p.ReadPost(postsIds[i])
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
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
		posts = append(posts, post)
	}
	return posts, err
}

func (p *Post) ReadPost(id int) (*types.Post, error) {
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
