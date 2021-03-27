package services

import (
	"encoding/binary"
	"io"
	"os"
	"time"
)

type PostIndex struct {
}

func NewPostIndex() (*PostIndex, error) {
	os.MkdirAll("db/posts-index/bytime/", os.ModePerm)
	return &PostIndex{}, nil
}

func (pt *PostIndex) Append(id int, ts time.Time) error {
	f, err := os.OpenFile("db/posts-index/bytime/"+ts.Format("2006-01-02"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	bId := Uint64ToByte(uint64(id))

	info, err := f.Stat()
	if err != nil {
		return err
	}

	_, err = f.WriteAt(bId, info.Size())
	if err != nil {
		return err
	}

	return err
}

func (pt *PostIndex) PostsByDay(ts time.Time) (ids []int64, err error) {
	f, err := os.OpenFile("db/posts-index/bytime/"+ts.Format("2006-01-02"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(b); i += 8 {
		ids = append(ids, int64(ByteToUint64(b[i:i+8])))
	}

	return ids, err
}

func (pt *PostIndex) PostsByRange(from, to time.Time) ([]int, error) {
	return nil, nil
}

func Uint64ToByte(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func ByteToUint64(v []byte) uint64 {
	return binary.BigEndian.Uint64(v)
}
