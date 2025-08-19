package test

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"testing"
)

func Test111(t *testing.T) {

	idList := []string{"1", "3", "3", "3", "3", "2", "4", "5", "6"}
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

func Test222(t *testing.T) {

	idList := []string{"1", "3", "3", "3", "3", "2", "2", "2", "4", "5", "6"}
	fmt.Println(FindDuplicates222(idList)) // 输出: [2 1 3]
}

func FindDuplicates222(ids []string) []string {

	restIds := make([]string, 0)
	existsMap := make(map[string]int)

	for _, id := range ids {
		val := existsMap[id]
		if val == 1 {
			restIds = append(restIds, id)
		}
		existsMap[id]++
	}

	return restIds
}

func Test333(t *testing.T) {
	movies := []string{"a", "b", "c", "d"}
	i := 1 // 删除 "b"

	movies = append(movies[:i], movies[i+1:]...)
	fmt.Println(movies) // [a c d]

	i = 2
	movies = append(movies[:i], movies[i+1:]...)
	fmt.Println(movies) // [a c]

	i = 0
	movies = append(movies[:i], movies[i+1:]...)
	fmt.Println(movies) // [c]

	i = 0
	movies = append(movies[:i], movies[i+1:]...)
	fmt.Println(movies) // []

}

func Test555(t *testing.T) {

	md5str, err := FileMD5("../mian.go")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("MD5:", md5str)

	sha1str, err := FileSha1("../mian.go")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("sha1str:", sha1str)

	sha256str, err := FileSHA256("../mian.go")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("SHA-256:", sha256str)

	sha3str, err := FileSHA512("../mian.go")
	if err != nil {
		fmt.Println("error", err)
		return
	}
	fmt.Println("SHA3-256:", sha3str)

}

// FileMD5 计算文件的 MD5 值
func FileMD5(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// 转成 16 进制字符串
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// FileSha1 计算文件的 sha1 值
func FileSha1(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// 转成 16 进制字符串
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func FileSHA256(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func FileSHA512(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha512.New512_256()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func Test666(t *testing.T) {
}
