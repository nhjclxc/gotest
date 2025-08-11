package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {

	var wg sync.WaitGroup

	wg.Add(1)
	doSomeThing(&wg)

	wg.Wait()

	fmt.Println("program is exited !!! ")

}

func doSomeThing(wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()

		// do something ...
		for i := 0; i < 5; i++ {
			fmt.Println("program running !!! ")
			time.Sleep(time.Second)
		}
	}()
}
