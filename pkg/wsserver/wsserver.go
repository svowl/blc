package wsserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"blc/pkg/auth"
	"blc/pkg/crawler"
	"blc/pkg/logger"
)

// Service это служба чата.
// - chMessages: ассоциативный массив каналов для записи сообщений во все открытые соединения по /messages
// - nextConnID: ID следущего соединения
// - logger:     интерфейс для записи логов
// - mux:        мьютекс нужен для установки лока при изменении общей памяти
type Service struct {
	upgrader      websocket.Upgrader
	crawlers      map[int]*crawler.Service
	nextCrawlerID int
	chMessages    map[int]chan string
	nextConnID    int
	nextErrConnID int
	logger        *logger.Logger
	mux           sync.Mutex
	router        *mux.Router
	auth          *auth.Auth
	delay         int
	chReport      chan *crawler.Service
}

// New возвращает новый объект службы
func New(logger *logger.Logger, crawlers map[int]*crawler.Service, r *mux.Router, a *auth.Auth, delay int, chReport chan *crawler.Service) *Service {
	var s Service
	s.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	s.crawlers = crawlers
	s.nextCrawlerID = 100
	s.chMessages = make(map[int]chan string)
	s.logger = logger
	s.router = r
	s.auth = a
	s.delay = delay
	s.chReport = chReport
	return &s
}

// Endpoints объявляет конечные точки
func (s *Service) Endpoints() {
	r := s.router //s.router.PathPrefix("/ws").Subrouter().StrictSlash(true)
	r.HandleFunc("/cmd/{token}", s.cmdHandler)
	r.HandleFunc("/messages/{token}", s.messagesHandler)
}

// PublishMessages отправляет сообщения всем подключенным клиентам
func (s *Service) PublishMessages(msgQueue chan crawler.ScanResult) {
	for msg := range msgQueue {
		encodedMsg, err := json.Marshal(msg)
		if err != nil {
			s.logger.Error(fmt.Sprintf("JSON encoding error: %v", err))
			continue
		}
		s.mux.Lock()
		for _, c := range s.chMessages {
			c <- string(encodedMsg)
		}
		s.mux.Unlock()
	}
}

// start получает список URL, стартует новый процесс сканирования и возвращает его идентификатор
func (s *Service) start(urls []string, depth int) int {
	if len(urls) == 0 {
		return 0
	}
	s.mux.Lock()
	ID := s.nextCrawlerID
	s.nextCrawlerID++
	s.crawlers[ID] = crawler.New(ID, s.delay, s.chReport, s.logger)
	s.mux.Unlock()

	go s.PublishMessages(s.crawlers[ID].ChResults)
	go func() {
		s.crawlers[ID].Scan(urls, depth, "", []string{})
	}()

	return ID
}

// Обработчик для /cmd принимает сообщение от пользователя
func (s *Service) cmdHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		s.logger.Error("/cmd: Ошибка соединения: " + err.Error())
		return
	}
	defer conn.Close()

	vars := mux.Vars(r)

	if ok := s.auth.ValidToken(vars["token"]); ok == false {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Unauthorized access"))
		s.logger.Error("/cmd: WriteMessage: Unauthorized access")
		return
	}
	s.logger.Info(fmt.Sprintf("/cmd: connection established: %s", vars["token"]))

	for {
		// Читаем сообщение
		mt, message, err := conn.ReadMessage()
		if err != nil {
			conn.WriteMessage(mt, []byte(err.Error()))
			s.logger.Error("/cmd: WriteMessage " + err.Error())
			return
		}

		s.logger.Info("/cmd: Command received: " + string(message))

		var cmdData struct {
			Cmd   string
			URLs  []string
			Depth int
			ID    int
		}
		if err := json.Unmarshal(message, &cmdData); err != nil {
			s.logger.Error("/cmd: Error: " + err.Error())
			continue
		}

		if cmdData.Cmd == "start" {
			s.start(cmdData.URLs, cmdData.Depth)
			s.logger.Info("/cmd: Start new process")
			continue
		}

		s.mux.Lock()
		if err := s.crawlers[cmdData.ID].Command(cmdData.Cmd); err != nil {
			s.logger.Error(fmt.Sprintf("/cmd[%d]: Error: %v", cmdData.ID, err))
		}
		s.mux.Unlock()
	}
}

// Обработчик для /messages пишет в соединение поток сообщений от crawler'а
func (s *Service) messagesHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		s.logger.Error("/messages: Ошибка соединения: " + err.Error())
		return
	}
	defer conn.Close()

	vars := mux.Vars(r)
	if ok := s.auth.ValidToken(vars["token"]); ok == false {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Unauthorized access"))
		s.logger.Error("/messages: WriteMessage: Unauthorized access")
		return
	}
	s.logger.Info(fmt.Sprintf("/messages: connection established: %s", vars["token"]))

	// Создаем новый канал для текущего соединения
	s.mux.Lock()
	connID := s.nextConnID
	s.nextConnID++
	connChannel := make(chan string)
	s.chMessages[connID] = connChannel
	s.mux.Unlock()

	defer func() {
		// Закрываем канал для соединения
		s.logger.Info(fmt.Sprintf("Завершено соединение %d", connID))

		s.mux.Lock()
		close(s.chMessages[connID])
		delete(s.chMessages, connID)
		s.mux.Unlock()
		s.logger.Info(fmt.Sprintf("Закрыт канал %d", connID))
	}()

	// Пишем в канал текущего соединения
	for message := range connChannel {
		err = conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			return
		}
		s.logger.Info(fmt.Sprintf("/messages: Write[conn %d]: %s", connID, message))
	}
}
