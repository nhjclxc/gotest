package os01_exec

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"os/exec"
	"testing"
)

/*
基本概念
os/exec 提供了 创建和运行外部进程的功能，主要类型和函数包括：

类型
exec.Cmd：表示一个外部命令。

函数

	exec.Command(name string, arg ...string) *exec.Cmd：创建一个命令对象。
	(*Cmd) Run() error：运行命令，等待其完成。
	(*Cmd) Start() error：启动命令，不等待完成。
	(*Cmd) Wait() error：等待命令完成（通常和 Start 配合）。
	(*Cmd) Output() ([]byte, error)：运行命令并返回标准输出。
	(*Cmd) CombinedOutput() ([]byte, error)：运行命令并返回标准输出+标准错误。
*/
func Test1(t *testing.T) {

	mvCmd := exec.Command("pwd") // can not use rename function in container enviornment, so use exec command
	//output, err := mvCmd.Output()
	//if err != nil {
	//	slog.Info("mv command err", slog.Any("err", err))
	//	return
	//}
	//slog.Info("mv command Output", slog.Any("output", output))
	// 在执行命令之前设置 stdout 和 stderr
	var stdout, stderr bytes.Buffer
	mvCmd.Stdout = &stdout
	mvCmd.Stderr = &stderr

	err2 := mvCmd.Run()
	if err2 != nil {
		slog.Info("mv command err", slog.Any("err2", err2))
	}

	// 记录 stdout 和 stderr 的内容
	if stdout.Len() > 0 {
		slog.Info("mv command stdout", slog.String("stdout", stdout.String()))
	}
	if stderr.Len() > 0 {
		slog.Warn("mv command stderr", slog.String("stderr", stderr.String()))
	}

}

func Test2(t *testing.T) {
	cmd := exec.Command("ping", "-c", "5", "www.baidu.com")

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	// 启动命令
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	// 异步读取 stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Printf("[STDOUT] %s\n", scanner.Text())
		}
	}()

	// 异步读取 stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Printf("[STDERR] %s\n", scanner.Text())
		}
	}()

	// 等待命令完成
	if err := cmd.Wait(); err != nil {
		fmt.Printf("命令执行失败: %v\n", err)
	}
}

func Test3(t *testing.T) {

	num := 123456

	fmt.Printf("%d \n", num)
	fmt.Printf("%3d \n", num)
	fmt.Printf("%6d \n", num)
	fmt.Printf("%9d \n", num)
}
