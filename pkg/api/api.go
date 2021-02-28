package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"blc/pkg/auth"
	"blc/pkg/crawler"
	"blc/pkg/logger"
	"blc/pkg/report"
)

// Service это служба Web-приложения, содержит ссылки на объекты роутера, БД и индекса
type Service struct {
	router   *mux.Router
	crawlers map[int]*crawler.Service
	auth     *auth.Auth
	logger   *logger.Logger
}

// New создает объект Service, объявляет endpoints
func New(logger *logger.Logger, crawlers map[int]*crawler.Service, r *mux.Router, a *auth.Auth) *Service {
	var s Service
	s.router = r
	s.crawlers = crawlers
	s.logger = logger
	s.router = r
	s.auth = a

	return &s
}

// Endpoints assigns API endpoints
func (s *Service) Endpoints() {
	r := s.router.PathPrefix("/api").Subrouter().StrictSlash(true)
	r.HandleFunc("/reports/{token}", s.reportsHandler).Methods(http.MethodPost)
	r.HandleFunc("/report/{token}", s.reportHandler).Methods(http.MethodPost)
	r.HandleFunc("/processerrors/{id}/{token}", s.processErrorsHandler).Methods(http.MethodPost)
	r.HandleFunc("/test/{token}", s.testTokenHandler).Methods(http.MethodGet)
	r.HandleFunc("/config", s.configHandler).Methods(http.MethodGet)
	r.HandleFunc("/signin", s.signinHandler).Methods(http.MethodPost)
}

// HTTP-handler api/test/{token} check token and returns "ok" on success
func (s *Service) testTokenHandler(w http.ResponseWriter, r *http.Request) {
	var result string = "ok"
	vars := mux.Vars(r)
	if ok := s.auth.ValidToken(vars["token"]); ok == false {
		result = "error"
	}

	if _, err := w.Write([]byte(result)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HTTP-handler api/signin receives user authentication data and returns token on success sign in
func (s *Service) signinHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Login, Password string
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	token, err := s.auth.SignIn(input.Login, input.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
	}
	if _, err := w.Write([]byte(token)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HTTP-handler api/reports/{token} returns JSON encoded list of available reports
func (s *Service) reportsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if ok := s.auth.ValidToken(vars["token"]); ok == false {
		http.Error(w, "Unauthorized access", http.StatusUnauthorized)
		s.logger.Error("/api/reports: Unauthorized access")
		return
	}
	reports, err := report.Reports()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	encoded, err := json.Marshal(reports)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if _, err := w.Write([]byte(encoded)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HTTP-handler api/report/{token} returns JSON encoded list of available reports
func (s *Service) reportHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if ok := s.auth.ValidToken(vars["token"]); ok == false {
		http.Error(w, "Unauthorized access", http.StatusUnauthorized)
		s.logger.Error("/api/reports: Unauthorized access")
		return
	}
	var reportDate string
	if err := json.NewDecoder(r.Body).Decode(&reportDate); err != nil {
		s.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	report, err := report.Report(reportDate)
	if err != nil {
		s.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if _, err := w.Write([]byte(report)); err != nil {
		s.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HTTP-handler api/processerrors/{id}/{token} returns JSON encoded list of process errors
func (s *Service) processErrorsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if ok := s.auth.ValidToken(vars["token"]); ok == false {
		http.Error(w, "Unauthorized access", http.StatusUnauthorized)
		s.logger.Error("/api/processerrors: Unauthorized access")
		return
	}
	_, ok := vars["id"]
	if ok == false {
		http.Error(w, "Process ID is empty", http.StatusInternalServerError)
		s.logger.Error("/api/processerrors: Process ID is empty")
		return
	}
	ID, err := strconv.Atoi(vars["id"])
	if err != nil {
		s.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	crw, ok := s.crawlers[ID]
	if ok == false {
		s.logger.Error(fmt.Sprintf("Process with specified ID (%v) not found", vars["id"]))
		http.Error(w, "Process with specified ID not found", http.StatusInternalServerError)
	}
	encoded, err := json.Marshal(crw.Errors)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if _, err := w.Write([]byte(encoded)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HTTP-handler api/config returns JSON encoded list of available reports
func (s *Service) configHandler(w http.ResponseWriter, r *http.Request) {
	data := [1]string{"ok"}

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
