package main

import (
	"fmt"
	"time"
)

func main() {

	var exitCh chan bool = make(chan bool)

	go doSomeThing(exitCh)

	<-exitCh

	fmt.Println("program is exited !!! ")

}

func doSomeThing(ch chan bool) {
	defer func() {
		ch <- true
	}()

	// do something ...
	for i := 0; i < 5; i++ {
		fmt.Println("program running !!! ")
		time.Sleep(time.Second)
	}
}
