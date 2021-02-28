// + build integration

package crawler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"testing"

	"blc/pkg/logger"
)

var host = ""

func TestMain(m *testing.M) {
	ts := httptest.NewServer(http.FileServer(http.Dir("./test")))
	defer ts.Close()
	host = ts.URL

	os.Exit(m.Run())
}

func TestService_parse(t *testing.T) {

	// Тестовая страница заведена специально для тестирования пакета spider,
	// имеет постоянную структуру и никогда не меняется.
	urls := []string{host + "/test/"}
	//urls := []string{"http://svowl.github.io/test/"}

	chReport := make(chan *Service)

	l := logger.New(os.Stdout, os.Stderr)

	s := New(1, 100, chReport, l)

	scanErrors := make(map[string]string)

	go func() {
		for {
			<-chReport
		}
	}()
	go func() {
		for v := range s.ChResults {
			if v.URL != "" && v.Error != "" {
				scanErrors[v.URL] = v.Error
			}
		}
	}()
	s.Scan(urls, -1, "", []string{host + "/test/not_existing.png"})

	close(s.ChResults)

	got := make([]string, 0, 14)
	for u := range s.Processed {
		got = append(got, u)
	}

	sort.Strings(got)

	want := []string{
		host + "/test/",
		host + "/test/gopher.jpg",
		host + "/test/iframe.html",
		host + "/test/not-existing.css",
		host + "/test/not_existing.js",
		// host + "/test/not_existing.png", - added into the exceptions
		host + "/test/not_existing_link.html",
		host + "/test/script.js",
		host + "/test/style.css",
		host + "/test2.html",
		host + "/test/../test3.html",
		host + "/test4.html",
		host + "/test/test/test5.html",
		host + "/test/?flags",
		"https://google.com",
	}
	sort.Strings(want)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Processed:\r\nполучено: %v\r\nожидается: %v", got, want)
	}

	wantErr := map[string]string{
		host + "/test/not-existing.css": "404 Not Found",
		host + "/test/not_existing.js":  "404 Not Found",
		//host + "/test/not_existing.png":       "404 Not Found", - added into the exceptions
		host + "/test/not_existing_link.html": "404 Not Found",
	}

	if !reflect.DeepEqual(scanErrors, wantErr) {
		t.Errorf("Errors:\r\nполучено: %v,\r\nожидается: %v", scanErrors, wantErr)
	}
}
