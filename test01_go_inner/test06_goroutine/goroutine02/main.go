package main

import (
	"fmt"
	"sync"
	"time"
)

/*
这个问题问得好 👍

在 Go 里：

```go
select {}
```

是一个 **永远阻塞的语句**。

---

### 为什么会这样？

* `select` 语句需要等待某个分支能执行。
* 如果 `select` 里有多个 `case`，会等待某个 channel 就绪。
* 如果写成 `select {}`，**没有任何 case 分支**，Go 就永远等不到分支可以执行，所以它会永久阻塞，不会退出。

换句话说，它相当于一个 **死循环阻塞**，不会占用 CPU：

```go

	for {
	    // do nothing
	}

```

但 `for {}` 会不停运行，占用一个 CPU 核心；而 `select {}` 内部是阻塞机制，不会占用 CPU 时间。

---

### 常见用法

 1. **保持 main() 不退出**
    在你写的并发程序里，通常调度器、worker 等都是跑在 goroutine 里。
    如果 main 函数退出，整个程序就退出了，所以需要一个“阻塞 main 的方法”。

    ```go
    func main() {
    go worker()
    select {} // 阻塞，防止 main 退出
    }
    ```

 2. **模拟永远阻塞的 goroutine**
    比如有些地方不想让 goroutine 结束，可以写：

    ```go
    go func() {
    defer fmt.Println("不会执行到这里")
    select {}
    }()
    ```

---

### 总结

* `select {}` 的作用：**让 goroutine 永久阻塞，且不占 CPU**。
* 常用在 `main()` 里，避免程序退出。

---

要不要我帮你写几个替代方案（比如 `sync.WaitGroup` 或 `signal.Notify` 方式），这样就能根据情况选择合适的“阻塞主进程”的方法？
*/
func main() {
	//go worker()
	//select {} // 阻塞，防止 main 退出

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		worker()
	}()

	wg.Wait() // 等待 worker 完成
}

func worker() {
	for i := 0; i < 10; i++ {
		fmt.Println("i = ", i)
		time.Sleep(500 * time.Millisecond)
	}
}
