package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

func main() {
	go messages()
	input()
}

// input ожидает ввод сообщения с клавиатуры и вызывает его отправку на сервер чата
func input() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		message, _ := reader.ReadString('\n')
		message = strings.Replace(message, "\r\n", "", -1)
		message = strings.Replace(message, "\n", "", -1)

		if message != "" {
			send(message)
		}
	}
}

// send отправляет сообщение на сервер чата по протоколу:
// client: "password"
// server: "OK"
// client: message <close connection>
func send(message string) {
	// Создаем соединение
	wsURL := "ws://localhost:8080/cmd"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("не удалось подключиться к серверу %s: %v", wsURL, err)
	}
	defer ws.Close()

	// Пишем в соединение "password"
	err = ws.WriteMessage(websocket.TextMessage, []byte("password"))
	if err != nil {
		log.Fatalf("Ошибка записи в соединение: %v", err)
	}

	// Читаем ответ сервера, ожидается, что это будет "OK"
	_, okMsg, err := ws.ReadMessage()
	if err != nil {
		log.Fatalf("не удалось прочитать сообщение: %v", err)
	}

	if string(okMsg) != "OK" {
		log.Fatalf("От сервера получен неожиданный ответ: %v", string(okMsg))
	}

	// Пишем в соединение сообщение message
	err = ws.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Fatalf("Ошибка записи в соединение: %v", err)
	}

	// Закрываем соединение
	err = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Fatalf("Ошибка закрытия соединения: %v", err)
	}
}

// messages устанавливает соединение с /messages и выводит получаемые сообщения в консоль
func messages() {
	// Создаем соединение
	wsURL := "ws://localhost:8080/messages"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("не удалось подключиться к серверу %s: %v", wsURL, err)
	}
	defer ws.Close()

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Fatalf("Ошибка чтения из соединения: %v", err)
		}
		fmt.Printf("\r%s\r\n> ", string(message))
	}
}
