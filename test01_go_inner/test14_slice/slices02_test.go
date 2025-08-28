package test14_slice

import (
	"cmp"
	"fmt"
	"slices"
	"testing"
)

// slices 包方法的使用

func Test0201(t *testing.T) {

	users := getUsers()

	// 在 slices.SortFunc 的比较函数里：
	//负数：a < b，顺序正确，不交换
	//正数：a > b，顺序错误，交换
	//0：相等，不交换

	// 年龄升序，成绩降序，名字按字典升序
	slices.SortFunc(users, func(a, b User) int {
		//if a.Age < b.Age {
		//	return cmp.Compare(a.Age, b.Age)
		//	//return -1
		//}
		//if a.Age > b.Age {
		//	return cmp.Compare(b.Age, a.Age)
		//	//return 1
		//}
		//return 0

		if r := cmp.Compare(a.Age, b.Age); r != 0 {
			return r // Age 升序
		}
		if r := cmp.Compare(b.Score, a.Score); r != 0 {
			return r // Score 降序（交换顺序）
		}
		return cmp.Compare(a.Name, b.Name) // Name 升序
	})

	for _, u := range users {
		fmt.Println(u)
	}

}

func getUsers() []User {
	users := []User{
		{"Tom", 20, 90},
		{"Alice", 18, 95},
		{"Bob", 20, 85},
		{"cat", 10, 88},
		{"David", 20, 79},
		{"Jack", 30, 88},
		{"Jack Chen", 60, 99},
	}
	return users
}

/*
📌 slices 常用方法清单 1. 排序相关

slices.Sort(slice)
按升序排序（元素必须是有序类型，如 int, string 等）。

slices.SortFunc(slice, func(a, b T) int)
自定义排序规则（适合结构体）。

slices.SortStableFunc(slice, func(a, b T) int)
稳定排序（保持相等元素的原始顺序）。

slices.IsSorted(slice)
判断是否已排序。

slices.IsSortedFunc(slice, cmp)
自定义比较函数，判断是否已排序。
*/

/*
📌 slices 常用方法清单 2. 搜索相关
slices.BinarySearch
*/
func Test0202(t *testing.T) {

	users := getUsers()

	_ = users

	nums := []int{1, 5, 66, 7, 8, 9}

	index, ok := slices.BinarySearch(nums, 10)
	fmt.Println(index)
	fmt.Println(ok)
	u := User{"Jack", 30, 881}
	index1, ok1 := slices.BinarySearchFunc(users, u, func(a User, b User) int {
		if r := cmp.Compare(a.Name, b.Name); r != 0 {
			return r
		}
		if r := cmp.Compare(a.Age, b.Age); r != 0 {
			return r
		}
		return cmp.Compare(a.Score, b.Score)
	})
	fmt.Println(index1)
	fmt.Println(ok1)

}

/*
3. 比较与检查

slices.Equal(a, b)
判断两个切片是否相等（元素逐一比较）。

slices.EqualFunc(a, b, eq)
使用自定义比较函数。

slices.Compare(a, b)
按字典序比较两个切片。返回 -1 / 0 / 1。

slices.CompareFunc(a, b, cmp)
自定义比较函数。
*/
func Test0203(t *testing.T) {

	nums1 := []int{1, 3, 5, 6, 8, 9}
	nums2 := []int{1, 3, 5, 6, 8, 9}

	fmt.Println(slices.Equal(nums1, nums2))
	fmt.Println(slices.Compare(nums1, nums2))

}

func Test0205(t *testing.T) {

	nums1 := []int{1, 3, 5, 7, 9}
	nums2 := []int{0, 2, 4, 6, 8}

	fmt.Println(merge(nums1, nums2))
	//fmt.Println(mergeSort(nums1, nums2))
}

func merge[T any](nums1, nums2 []T) []T {
	return append(nums1, nums2...)
}

//func mergeSort[T constraints.Ordered](nums1, nums2 []T) []T {
//	nums := merge(nums1, nums2)
//	slices.SortFunc(nums, func(a, b T) int {
//		return a - b
//	})
//	return nums
//}

func Test0206(t *testing.T) {
	// 💡 总结：
	//[...]T{} → 数组，长度自动推导，不能扩容
	//[]T{} → 切片，长度可变，可 append

	var arr1 = [5]int{1, 2, 3, 4, 5}
	var arr2 = [5]int{1, 2, 3}
	var arr3 = [...]int{1, 2, 3, 4, 5}
	var arr5 = []int{1, 2, 3, 4, 5}

	fmt.Printf(" arr1 = %#v, len = %d, cap = %d \n", arr1, len(arr1), cap(arr1))
	fmt.Printf(" arr2 = %#v, len = %d, cap = %d \n", arr2, len(arr2), cap(arr2))
	fmt.Printf(" arr3 = %#v, len = %d, cap = %d \n", arr3, len(arr3), cap(arr3))
	fmt.Printf(" arr5 = %#v, len = %d, cap = %d \n", arr5, len(arr5), cap(arr5))

	//arr1 = append(arr1, 8) // 无法将 'arr1' (类型 [5]int) 用作类型 []Type
	arr5 = append(arr5, 8)

}
