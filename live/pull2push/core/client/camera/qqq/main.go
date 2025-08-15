package main

import (
	"fmt"
	"io"
	"log"
)

func main() {
	var data []byte = make([]byte, 10)
	data[0] = 0
	data[1] = 1
	data[2] = 2
	data[3] = 3
	data[4] = 4
	data[5] = 5
	data[6] = 6
	data[7] = 7
	data[8] = 8
	data[9] = 9
	err := safeWriteChunks(nil, data, 51)
	if err != nil {
		log.Println("发送错误:", err)
	}

}

func safeWriteChunks(w io.Writer, data []byte, chunkSize int) error {
	total := len(data)
	offset := 0

	for offset < total {
		remaining := total - offset
		size := chunkSize
		if remaining < chunkSize {
			size = remaining
		}

		n, err := w.Write(data[offset : offset+size])
		if err != nil {
			return fmt.Errorf("写入错误: %w", err)
		}
		if n != size {
			return fmt.Errorf("短写: 期望 %d, 实际 %d", size, n)
		}

		offset += size
	}

	return nil
}

func Write(bytes []byte) (int, error) {
	for _, val := range bytes {
		fmt.Println(val)
	}
	return len(bytes), nil
}
