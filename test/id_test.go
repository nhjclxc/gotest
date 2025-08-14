package test

import (
	"fmt"
	"testing"
)

func Test111(t *testing.T) {

	idList := []string{"1", "3", "3", "2", "4", "5", "6"}
	fmt.Println(FindDuplicates(idList)) // 输出: [2 1 3]
}

// FindDuplicates 找出切片中重复的元素
func FindDuplicates(idList []string) []string {
	countMap := make(map[string]int)
	var duplicates []string
	seen := make(map[string]bool) // 用于避免重复元素被多次加入结果

	for _, id := range idList {
		countMap[id]++
		if countMap[id] > 1 && !seen[id] {
			duplicates = append(duplicates, id)
			seen[id] = true
		}
	}

	return duplicates
}
