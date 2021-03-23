package services

import (
	"encoding/json"
	"github.com/TokDenis/micro-blog/types"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"strconv"
	"time"
)

type Stats struct {
	viewsChan chan int
}

func NewStats() *Stats {
	s := Stats{
		viewsChan: make(chan int, 1000),
	}
	go s.viewsCollector()

	return &s
}

func (s *Stats) CreateStats(postId int) error {
	f, err := os.OpenFile("db/stats/"+strconv.Itoa(postId), os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return err
	}

	defer f.Close()

	b, err := json.Marshal(types.Stats{
		Id:    postId,
		Views: 0,
	})

	f.Write(b)
	return err
}

func (s *Stats) addViews(postId, count int) error {
	f, err := os.OpenFile("db/stats/"+strconv.Itoa(postId), os.O_RDWR, 0777)
	if err != nil {
		return err
	}

	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	var stats types.Stats

	err = json.Unmarshal(b, &stats)
	if err != nil {
		return err
	}

	stats.Views += int64(count)

	err = f.Truncate(int64(len(b)))
	if err != nil {
		return err
	}

	b, err = json.Marshal(&stats)
	if err != nil {
		return err
	}

	_, err = f.WriteAt(b, 0)
	if err != nil {
		return err
	}
	return err
}

func (s *Stats) CountView(postId int) {
	s.viewsChan <- postId
}

func (s *Stats) viewsCollector() {
	viewsMap := make(map[int]int)
	tic := time.NewTicker(time.Minute)
	for postId := range s.viewsChan {
		viewsMap[postId]++

		select {
		case <-tic.C:
			for postId, count := range viewsMap {
				err := s.addViews(postId, count)
				if err != nil {
					log.Error().Err(err).Send()
				}
				delete(viewsMap, postId)
			}
		default:

		}
	}
}

func (s *Stats) ReadStats(postId int) (*types.Stats, error) {
	b, err := os.ReadFile("db/stats/" + strconv.Itoa(postId))
	if err != nil {
		return nil, err
	}

	var stats types.Stats

	err = json.Unmarshal(b, &stats)
	if err != nil {
		return nil, err
	}

	return &stats, err
}

func (s *Stats) TopReadPostsStats(posts []int) (map[int]*types.Stats, error) {
	stats := make(map[int]*types.Stats)

	for i := 0; i < len(posts); i++ {
		stat, err := s.ReadStats(posts[i])
		if err != nil {
			return nil, err
		}

		stats[stat.Id] = stat
	}

	return stats, nil
}
