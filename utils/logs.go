package utils

import "fmt"

func Error(err error) {
	fmt.Println(fmt.Sprintf("[E] %s", err.Error()))
}
func Info(msg string) {
	fmt.Println(fmt.Sprintf("[I] %s", msg))
}
