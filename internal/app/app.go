package app

import (
	"fmt"
	"io"
	"net/http"
)

func Run() error {
	res, _ := http.Get("http://sidingkas:5995/api/v2/status")
	// Считываем содержимое тела ответа в буфер
	b, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	// Отправляем содержимое тела ответа в стандартный поток вывода
	fmt.Printf("%s", b)
	return nil
}
