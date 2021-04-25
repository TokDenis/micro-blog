package services

import (
	"fmt"
	"testing"
)

func TestUnValidatePost(t *testing.T) {
	p := Post{validPostIds: []int{0, 1, 2, 3, 4, 5}}

	p.unValidPost(2)

	if fmt.Sprint(p.validPostIds) != "[0 1 3 4 5]" {
		t.Error("not valid validPostIds")
	}

	p.unValidPost(3)

	if fmt.Sprint(p.validPostIds) != "[0 1 4 5]" {
		t.Error("not valid validPostIds")
	}

	p.addValidPost(3)

	if fmt.Sprint(p.validPostIds) != "[0 1 3 4 5]" {
		t.Error("not valid validPostIds")
	}

	p.addValidPost(6)
	p.addValidPost(7)
	p.addValidPost(8)

	if fmt.Sprint(p.validPostIds) != "[0 1 3 4 5 6 7 8]" {
		t.Error("not valid validPostIds")
	}

	p.unValidPost(7)

	if fmt.Sprint(p.validPostIds) != "[0 1 3 4 5 6 8]" {
		t.Error("not valid validPostIds")
	}
}
