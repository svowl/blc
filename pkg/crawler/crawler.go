package crawler

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"golang.org/x/net/html"

	"blc/pkg/logger"
)

// Service это служба поискового робота
type Service struct {
	// Список просканированных сайтов
	Processed map[string]bool
	// Канал для результатов сканирования
	ChResults chan ScanResult
	Errors    map[string]ErrorResult
	// Текущее состояние
	currentState int
	// Команда
	Cmd int
	// Задержка, ms
	Delay        int
	ID           int
	chReport     chan *Service
	URLs         []string
	excludedURLs map[string]bool
	logger       *logger.Logger
	TimeElapsed  time.Duration
	TimeFinished time.Time
	sessionName  string
	cookies      map[string][]*http.Cookie
}

// Статусы процесса
const (
	// STOPPED - процесс не начат/остановлен
	STOPPED = iota + 1
	// INPROGRESS - процесс идет
	INPROGRESS
	// PAUSED - процесс приостановлен
	PAUSED
)

// Типы командных сообщений
const (
	// PAUSE - поставить процесс сканирования на паузу
	PAUSE = iota + 100
	// PROCEED - возобновить процесс сканирования
	PROCEED
	// CANCEL Прервать процесс сканирования
	CANCEL
)

// ScanResult это структура, описывающая формат данных об результате сканирования
type ScanResult struct {
	URL           string
	State         int
	HTTPStatus    int
	Error         string
	ParentURL     string
	ProgressState int
	ID            int
	TotalLinks    int
	TotalErrors   int
	URLs          []string
}

// ErrorResult это структура, описывающая формат данных об ошибке сканирования
type ErrorResult struct {
	HTTPStatus int
	Error      string
	ParentURL  string
	depth      int
}

// New возвращает новый объект службы поискового робота
func New(ID int, delay int, chReport chan *Service, logger *logger.Logger) *Service {
	var s Service
	s.ID = ID
	s.Processed = make(map[string]bool)
	s.ChResults = make(chan ScanResult)
	s.Errors = make(map[string]ErrorResult)
	s.currentState = STOPPED
	s.Cmd = 0
	s.Delay = delay
	s.chReport = chReport
	s.URLs = make([]string, 0, 2)
	s.excludedURLs = make(map[string]bool)
	s.cookies = make(map[string][]*http.Cookie)
	s.logger = logger
	return &s
}

// Command принимает команду в строковом виде
func (s *Service) Command(cmd string) error {
	switch strings.ToUpper(cmd) {
	case "PAUSE":
		s.Cmd = PAUSE
	case "PROCEED":
		s.Cmd = PROCEED
	case "CANCEL":
		s.Cmd = CANCEL
	default:
		return errors.New("Unknown crawler command: " + cmd)
	}
	s.logger.Info(fmt.Sprintf("Command %v %v", cmd, s.Cmd))
	s.updateState()
	return nil
}

// updateState обновляет текущее состояние процесса
func (s *Service) updateState() {
	if s.Cmd == 0 {
		return
	}
	switch s.Cmd {
	case PAUSE:
		if s.currentState == INPROGRESS {
			s.currentState = PAUSED
		}
	case PROCEED:
		if s.currentState == PAUSED {
			s.currentState = INPROGRESS
		}
	case CANCEL:
		s.currentState = STOPPED
	}
	s.ChResults <- ScanResult{ProgressState: s.currentState, ID: s.ID, TotalLinks: len(s.Processed), TotalErrors: len(s.Errors)}
	s.Cmd = 0
}

// Scan запускает сканирование сайта
// Параметры:
// - url: URL сайта,
// - depth: глубина сканирования (-1 снимает ограничение),
// - cmd: канал с командами
// Возвращает канал с успешными ссылками и канал с ошибками.
func (s *Service) Scan(urls []string, depth int, sessionName string, excludedURLs []string) {
	started := time.Now()
	s.logger.Info(fmt.Sprintf("Started, ID: %d...", s.ID))
	s.currentState = INPROGRESS
	s.ChResults <- ScanResult{ProgressState: s.currentState, ID: s.ID, TotalLinks: len(s.Processed), TotalErrors: len(s.Errors)}
	s.sessionName = sessionName
	if len(excludedURLs) > 0 {
		for _, u := range excludedURLs {
			s.excludedURLs[u] = true
		}
	}
	for _, url := range urls {
		s.URLs = append(s.URLs, url)
		s.ChResults <- ScanResult{ProgressState: s.currentState, ID: s.ID, TotalLinks: len(s.Processed), TotalErrors: len(s.Errors), URLs: s.URLs}
		s.parse(url, url, depth)
	}
	// Re-scan errors
	var savedDelay int
	savedDelay, s.Delay = s.Delay, 3
	for u, e := range s.Errors {
		// Re-scan only errors with code 0 and >= 500
		if e.HTTPStatus > 0 && e.HTTPStatus < 500 {
			continue
		}
		s.logger.Info(fmt.Sprintf("Re-scan %s", u))
		s.parse(u, e.ParentURL, e.depth)
	}
	s.Delay = savedDelay
	s.currentState = STOPPED
	s.ChResults <- ScanResult{ProgressState: s.currentState, ID: s.ID, TotalLinks: len(s.Processed), TotalErrors: len(s.Errors), URLs: s.URLs}
	s.logger.Info(fmt.Sprintf("Finished, ID: %d...", s.ID))
	s.TimeFinished = time.Now()
	s.TimeElapsed = s.TimeFinished.Sub(started)

	s.chReport <- s
}

func (s *Service) parse(link string, baseLink string, depth int) {

	if len(s.Errors) > 35 {
		s.currentState = STOPPED
		return
	}

	if s.currentState == PAUSED {
		start := time.Now()
		for s.currentState == PAUSED {
			// Ожидание завершения паузы
			time.Sleep(time.Second * 1)
			// Каждые 5 секунд пишем в канал результатов текущее состояние
			if int(time.Since(start).Seconds())%5 == 0 {
				s.ChResults <- ScanResult{ProgressState: s.currentState, ID: s.ID, TotalLinks: len(s.Processed), TotalErrors: len(s.Errors), URLs: s.URLs}
			}
			// Через час снимаем с паузы и завершаем процесс
			if int(time.Since(start).Hours()) >= 1 {
				s.currentState = STOPPED
				break
			}
		}
	}

	if s.currentState == STOPPED {
		return
	}

	if depth == 0 {
		return
	}

	// Delay
	time.Sleep(time.Millisecond * time.Duration(s.Delay))

	s.Processed[link] = true

	// To skip SSL certificate issues
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	// Set client's timeout and transport
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: tr,
	}
	// Detect request method (GET or HEAD)
	method := method(link)

	// Make request
	request, err := http.NewRequest(method, link, nil)
	if err != nil {
		s.Errors[link] = ErrorResult{HTTPStatus: 0, Error: fmt.Sprintf("%v", err), ParentURL: baseLink}
		s.ChResults <- ScanResult{URL: link, HTTPStatus: 0, Error: fmt.Sprintf("%v", err), ParentURL: baseLink, ProgressState: s.currentState, ID: s.ID, TotalLinks: len(s.Processed), TotalErrors: len(s.Errors), URLs: s.URLs}
		return
	}

	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.104 Safari/537.36")
	request.Header.Add("Accept", "*/*")
	//request.Header.Add("Accept-Encoding", "gzip, deflate, br")
	request.Header.Add("Connection", "keep-alive")

	parsedLink, err := url.Parse(link)
	if err != nil {
		return
	}
	host := parsedLink.Hostname()

	if cookies, ok := s.cookies[host]; ok == true {
		for _, c := range cookies {
			request.AddCookie(c)
		}
	}
	response, err := client.Do(request)
	if err != nil {
		s.Errors[link] = ErrorResult{HTTPStatus: 0, Error: fmt.Sprintf("%s error: %v", method, err), ParentURL: baseLink}
		s.ChResults <- ScanResult{URL: link, HTTPStatus: 0, Error: fmt.Sprintf("%s error: %v", method, err), ParentURL: baseLink, ProgressState: s.currentState, ID: s.ID, TotalLinks: len(s.Processed), TotalErrors: len(s.Errors), URLs: s.URLs}
		return
	}
	defer response.Body.Close()

	if response.StatusCode == 403 && response.Header.Get("Cf-Chl-Bypass") == "1" {
		err := "Protected by CloudFlare CAPTCHA"
		s.Errors[link] = ErrorResult{HTTPStatus: response.StatusCode, Error: err, ParentURL: baseLink, depth: depth}
		s.ChResults <- ScanResult{URL: link, HTTPStatus: response.StatusCode, Error: err, ParentURL: baseLink, ProgressState: s.currentState, ID: s.ID, TotalLinks: len(s.Processed), TotalErrors: len(s.Errors), URLs: s.URLs}
		return
	}

	if response.StatusCode > 400 && response.StatusCode != 418 {
		s.Errors[link] = ErrorResult{HTTPStatus: response.StatusCode, Error: response.Status, ParentURL: baseLink, depth: depth}
		s.ChResults <- ScanResult{URL: link, HTTPStatus: response.StatusCode, Error: response.Status, ParentURL: baseLink, ProgressState: s.currentState, ID: s.ID, TotalLinks: len(s.Processed), TotalErrors: len(s.Errors), URLs: s.URLs}
		return
	}

	// Success
	s.ChResults <- ScanResult{URL: link, State: 1, HTTPStatus: response.StatusCode, ProgressState: s.currentState, ID: s.ID, TotalLinks: len(s.Processed), TotalErrors: len(s.Errors), URLs: s.URLs}
	if _, ok := s.Errors[link]; ok {
		delete(s.Errors, link)
	}

	if depth == 1 {
		return
	}

	docType := response.Header.Get("Content-type")
	if !strings.Contains(docType, "text/html") {
		return
	}

	// Парсим только если вернулся HTML
	page, err := html.Parse(response.Body)
	if err != nil {
		// Не смогли распарсить, ну и ладно, выходим
		return
	}

	if s.sessionName != "" {
		if _, ok := s.cookies[host]; ok == false {
			for _, c := range response.Cookies() {
				if c.Name == s.sessionName {
					s.cookies[host] = append(s.cookies[host], c)
				}
			}
		}
	}

	// Парсим базовый URL
	base, err := url.Parse(baseLink)
	if err != nil {
		// Ошибка парсинга базового URL - странная ситуация, пропускаем ход, но пишем в канал ошибок
		s.Errors[link] = ErrorResult{Error: fmt.Sprintf("URL parse error: %v", err), ParentURL: baseLink, depth: depth}
		s.ChResults <- ScanResult{URL: link, Error: fmt.Sprintf("URL parse error: %v", err), ParentURL: baseLink, ProgressState: s.currentState, ID: s.ID, TotalLinks: len(s.Processed), TotalErrors: len(s.Errors), URLs: s.URLs}
		return
	}

	links := make(map[string]bool)
	baseURI := pageLinks(links, page)

	for l := range links {
		u, err := url.Parse(l)
		if err != nil {
			// Ошибка парсинга URL - пропускаем ссылку и продолжаем дальше
			continue
		}
		if u.IsAbs() == true {
			// Абсолютная ссылка - оставляем как есть

		} else if strings.HasPrefix(l, "//") {
			// Абсолютная ссылка вида "//foo", добавляем схему (http/https) из базового URL
			u.Scheme = base.Scheme

		} else if strings.HasPrefix(l, "/") {
			// Относительная ссылка вида "/foo", добаввляем схему и хост из базового URL
			u.Scheme = base.Scheme
			u.Host = base.Host

		} else {
			// Остальные ссылки считаем относительными от текущего пути в базовом URL: "foo", "./foo" etc
			// Добавляем схему, хост и path базового URL
			// т.е. если базовый URL http://example.com/foo/test.html, а текущая ссылка "bar.html"
			// ссылка будет превращена в http://example.com/foo/bar.html
			u.Scheme = base.Scheme
			u.Host = base.Host
			p := path.Clean(u.Path)
			if p == "." {
				p = ""
			}
			basePath := base.Path
			if baseURI != "" {
				if bURL, err := url.Parse(baseURI); err == nil {
					u.Scheme = bURL.Scheme
					u.Host = bURL.Host
					basePath = bURL.Path
				}
			}
			u.Path = strings.TrimRight(path.Dir(basePath), "/") + "/" + p
		}
		u.Fragment = ""
		newURL := u.String()
		// Ссылка уже отсканирована - пропускаем
		if _, found := s.Processed[newURL]; found {
			continue
		}
		if _, found := s.excludedURLs[newURL]; found {
			continue
		}
		newDepth := depth - 1
		// Сканируем ссылки с других хостов только на глубину 1
		if u.Host != base.Host {
			newDepth = 1
		}
		s.parse(newURL, link, newDepth)
	}
}

func pageLinks(links map[string]bool, n *html.Node) string {
	var base string
	tagsAttr := map[string]string{
		"a":      "href",
		"link":   "href",
		"script": "src",
		"img":    "src",
		"iframe": "src",
		"base":   "href",
	}
	prohibitedPrefixes := []string{"tel:", "mailto:", "javascript:"}
	if attr, ok := tagsAttr[n.Data]; ok && n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == attr {
				allowed := true
				for _, prefix := range prohibitedPrefixes {
					if strings.HasPrefix(a.Val, prefix) {
						allowed = false
					}
				}
				if !allowed {
					continue
				}
				if n.Data == "base" {
					base = a.Val
				}
				if _, found := links[a.Val]; !found {
					links[a.Val] = true
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		rbase := pageLinks(links, c)
		if rbase != "" {
			base = rbase
		}
	}
	return base
}

func method(link string) string {
	if u, err := url.Parse(link); err == nil {
		exts := map[string]bool{
			"":      true,
			".html": true,
			".htm":  true,
			".asp":  true,
			".aspx": true,
		}
		if _, ok := exts[ext(u.Path)]; ok == true {
			return "GET"
		}
	}

	return "HEAD"
}

func ext(fpath string) string {
	return path.Ext(fpath)
}
