package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {

	fmt.Println(111)

	var dataCh = make(chan byte)

	go send(dataCh)

	receive(dataCh)

}

func send(ch chan<- byte) {
	ticker := time.NewTicker(3 * time.Second)

	for {
		select {
		case <-ticker.C:
			data := byte(rand.Intn(127))
			ch <- data
			fmt.Println("send b = ", data)
		}
	}
}

func receive(ch <-chan byte) {
	for {
		select {
		case b := <-ch:
			fmt.Println("receive b = ", b)
		}
	}
}
