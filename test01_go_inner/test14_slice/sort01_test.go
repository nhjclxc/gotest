package test14_slice

import (
	"cmp"
	"fmt"
	"slices"
	"sort"
	"testing"
)

type User struct {
	Name  string
	Age   int
	Score int
}

func Test01(t *testing.T) {
	users := []User{
		{"Tom", 20, 90},
		{"Alice", 18, 95},
		{"Bob", 20, 85},
		{"David", 20, 85},
	}

	// 排序：Age 升序 -> Score 降序 -> Name 升序
	sort.Slice(users, func(i, j int) bool {
		if users[i].Age != users[j].Age {
			return users[i].Age < users[j].Age
		}
		if users[i].Score != users[j].Score {
			return users[i].Score > users[j].Score // 降序
		}
		return users[i].Name < users[j].Name
	})

	for _, u := range users {
		fmt.Println(u)
	}
}

func Test02(t *testing.T) {

	users := []User{
		{"Tom", 20, 90},
		{"Alice", 18, 95},
		{"Bob", 20, 85},
		{"David", 20, 85},
	}

	slices.SortFunc(users, func(a, b User) int {
		if a.Age != b.Age {
			return cmp.Compare(a.Age, b.Age) // 升序
		}
		if a.Score != b.Score {
			return cmp.Compare(b.Score, a.Score) // 降序
		}
		return cmp.Compare(a.Name, b.Name) // 升序
	})
	for _, u := range users {
		fmt.Println(u)
	}

}
