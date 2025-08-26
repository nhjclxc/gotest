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
	"time"
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

	//2025-08-20 14:15:00， 2027245.8 bytes/s, 33.2 Mb/s
	//fmt.Printf("Mb/s:   %.2f\n", (float64(2027245) * 8 / 1_000_000))
	//
	//fmt.Printf("Mb/s:   %.2f\n", (float64(4194310) * 8 / 1_000_000))
	//
	//fmt.Printf("Mb/s:   %.2f\n", (float64(4054491) * 8 / 1_000_000))
	//fmt.Printf("Mb/s:   %.2f\n", (float64(4124393.2) * 8 / 1_000_000))

	fmt.Printf("Mb/s:   %.2f\n", (float64(3635065) * 8 / 1_000_000))

	//2027243
}

func Test888(t *testing.T) {
	cst := time.FixedZone("CST", 8*3600)
	fmt.Println(cst)

	fmt.Println(time.Now())

}

func Test999(t *testing.T) {

	start := time.Date(2025, 8, 21, 9, 30, 0, 0, time.Local)
	end := time.Date(2025, 8, 21, 10, 0, 0, 0, time.Local)
	slots := genTimeSlot(start, end, 5)
	for _, s := range slots {
		fmt.Println(s)
	}
}

// genTimeSlot 生成从 startTime 到 endTime 的时间槽（按 segmentMinutes 分段），返回时间字符串切片
func genTimeSlot(startTime, endTime time.Time, segmentMinutes int) []string {
	var slots []string

	// 将 startTime 向上对齐到 segmentMinutes 的倍数
	min := (startTime.Minute()/segmentMinutes + 1) * segmentMinutes
	hour := startTime.Hour()
	day := startTime.Day()
	month := int(startTime.Month())
	year := startTime.Year()

	if min >= 60 {
		min -= 60
		hour += 1
		if hour >= 24 {
			hour = 0
			startTime = startTime.AddDate(0, 0, 1)
			year = startTime.Year()
			month = int(startTime.Month())
			day = startTime.Day()
		}
	}

	current := time.Date(year, time.Month(month), day, hour, min, 0, 0, startTime.Location())

	for !current.After(endTime) {
		slots = append(slots, current.Format("2006-01-02 15:04:05"))
		current = current.Add(time.Duration(segmentMinutes) * time.Minute)
	}

	return slots
}
