package test11_type

import (
	"crypto/md5"
	"fmt"
	"testing"
	"time"
)

type AddType struct {
	val int
}

func (a AddType) Add(num int) AddType {
	a.val += num
	return a
}

func (a AddType) String() string {
	return fmt.Sprintf("%d", a.val)
}

func Test1(t *testing.T) {

	var at AddType = AddType{val: 1}

	a := at.Add(1)
	fmt.Println(a)
	a = a.Add(1)
	fmt.Println(a)
}

func Test2(tt *testing.T) {
	t := time.Date(2025, 8, 26, 11, 27, 45, 0, time.Local)

	// 对齐到分钟
	aligned := t.Truncate(time.Minute)
	fmt.Println(aligned) // 2025-08-26 11:23:00

	// 对齐到 5 分钟
	aligned5 := t.Truncate(5 * time.Minute)
	fmt.Println(aligned5) // 2025-08-26 11:20:00

	// 四舍五入到分钟
	rounded := t.Round(time.Minute)
	fmt.Println(rounded) // 2025-08-26 11:24:00

	// 四舍五入到 5 分钟
	rounded5 := t.Round(5 * time.Minute)
	fmt.Println(rounded5) // 2025-08-26 11:25:00

	st := time.Date(2025, 8, 26, 10, 27, 45, 0, time.Local)
	et := time.Date(2025, 8, 26, 11, 27, 45, 0, time.Local)

	ss := genTimeSlot(st, et, 5)
	for _, s := range ss {
		fmt.Println(s)
	}
	fmt.Println()
	ss2 := genTimeSlot2(st, et, 5)
	for _, s := range ss2 {
		fmt.Println(s)
	}

}

func genTimeSlot2(startTime, endTime time.Time, segmentMinutes int) []string {
	var slots []string
	if segmentMinutes <= 0 {
		return slots
	}

	// 确保 startTime 在 loc 时区
	startTime = startTime.In(startTime.Location())
	endTime = endTime.In(startTime.Location())

	// 向上对齐到 segmentMinutes 的倍数
	current := startTime.Truncate(time.Duration(segmentMinutes) * time.Minute)
	if current.Before(startTime) {
		current = current.Add(time.Duration(segmentMinutes) * time.Minute)
	}

	for !current.After(endTime) {
		slots = append(slots, current.Format("2006-01-02 15:04:05"))
		current = current.Add(time.Duration(segmentMinutes) * time.Minute)
	}

	return slots
}

// genTimeSlot 生成从 startTime 到 endTime 的时间槽（按 segmentMinutes 分段），返回时间字符串切片
func genTimeSlot(startTime, endTime time.Time, segmentMinutes int) []string {
	var slots []string

	// 将 startTime 向上对齐到 segmentMinutes 的倍数
	min1 := (startTime.Minute()/segmentMinutes + 1) * segmentMinutes
	hour := startTime.Hour()
	day := startTime.Day()
	month := int(startTime.Month())
	year := startTime.Year()

	if min1 >= 60 {
		min1 -= 60
		hour += 1
		if hour >= 24 {
			hour = 0
			startTime = startTime.AddDate(0, 0, 1)
			year = startTime.Year()
			month = int(startTime.Month())
			day = startTime.Day()
		}
	}

	current := time.Date(year, time.Month(month), day, hour, min1, 0, 0, startTime.Location())

	for !current.After(endTime) {
		slots = append(slots, current.Format("2006-01-02 15:04:05"))
		current = current.Add(time.Duration(segmentMinutes) * time.Minute)
	}

	return slots
}

func Test3(tt *testing.T) {
	t1 := time.Date(2025, 8, 26, 11, 27, 45, 0, time.Local)
	t2 := time.Date(2025, 8, 26, 12, 27, 45, 0, time.Local)
	t3 := time.Date(2025, 8, 26, 12, 27, 45, 0, time.Local)
	t5 := t3

	fmt.Println("t1 ", t1.String())
	fmt.Println("t2 ", t2.String())
	fmt.Println("t3 ", t3.String())
	fmt.Println("t5 ", t5.String())

	fmt.Println("t1.Before(t2) ", t1.Before(t2)) // true
	fmt.Println("t1.After(t2) ", t1.After(t2))   // false
	fmt.Println("t2.Before(t1) ", t2.Before(t1)) // false
	fmt.Println("t2.After(t1) ", t2.After(t1))   // true
	fmt.Println("t2.Before(t3) ", t2.Before(t3)) // false
	fmt.Println("t2.After(t3) ", t2.After(t3))   // false
	fmt.Println("t2.After(t3) ", t2.Equal(t3))   // true
	fmt.Println("t3.After(t5) ", t3.Equal(t5))   // true

	fmt.Println()
	fmt.Println(t1.Date())
	fmt.Println(t1.Clock())
	fmt.Println(t1.Weekday())
	fmt.Println(t1.ISOWeek())

}

func Test5(ttt *testing.T) {
	t1 := time.Date(2025, 8, 26, 11, 27, 45, 0, time.Local)

	t1 = time.Now()
	t1Str := t1.Format("2006年01月02日 15时04分05秒")
	fmt.Println(t1Str)

	//t1Str = "2025年08月26日 11时27分45秒"

	parse, err := time.Parse("2006年01月02日 15时04分05秒", t1Str)
	if err != nil {
		return
	}
	fmt.Println(parse)

}

func Test6(t *testing.T) {
	reqTime := time.Now().Unix()
	fmt.Println("reqTime = ", reqTime)
	retoken := fmt.Sprintf("%x", md5.Sum([]byte("dns"+"Spazvb2P"+fmt.Sprintf("%d", int(reqTime)))))
	fmt.Println(retoken)
	retoken = fmt.Sprintf("%x", md5.Sum([]byte("dns"+"Spazvb2P"+fmt.Sprintf("%d", int(reqTime)))))
	fmt.Println(retoken)
	retoken = fmt.Sprintf("%x", md5.Sum([]byte("dns"+"Spazvb2P"+fmt.Sprintf("%d", int(reqTime)))))
	fmt.Println(retoken)

}
