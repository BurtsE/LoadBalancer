package server

import (
	"LoadBalancer/internal/config"
	"LoadBalancer/pkg/balancer"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type errorResponce struct {
	Error string `json:"message"`
	Code  int    `json:"code"`
}

type Balancer interface {
	GetNextURL() *url.URL
}

type Limiter interface {
	Allow(ip string) bool
}

type Server struct {
	srv      *http.Server
	proxy    *httputil.ReverseProxy
	balancer Balancer
	limiter  Limiter
}

func NewServer(config config.Config, balancer *balancer.Balancer, limiter Limiter) *Server {
	s := &Server{

		balancer: balancer,
		limiter:  limiter,
	}
	proxy := &httputil.ReverseProxy{
		Director:       s.director,
		ErrorHandler:   errorHandler,
		ModifyResponse: modifyResponse,
	}
	server := &http.Server{
		Addr:    ":" + config.ServerPort,
		Handler: http.HandlerFunc(s.handleRequest),
	}
	s.srv = server
	s.proxy = proxy
	return s
}

func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

// Обработчик запросов
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	clientIP := r.RemoteAddr
	if !s.limiter.Allow(clientIP) {
		writeErrorResponse(w, fmt.Errorf("rate limit exceeded"), http.StatusTooManyRequests)
		return
	}

	s.proxy.ServeHTTP(w, r)
	log.Printf("Request completed in %v", time.Since(start))
}

// Настройка прокси
func (s *Server) director(req *http.Request) {
	backend := s.balancer.GetNextURL()
	if backend == nil {
		return
	}
	log.Printf("Forwarding request to %s | %s %s", backend.Host, req.Method, req.URL.Path)
	req.URL.Scheme = backend.Scheme
	req.URL.Host = backend.Host
	req.Header.Set("X-Forwarded-For", req.RemoteAddr)
}

// Обработка ошибок бэкенда
func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("Backend error: %v", err)
	writeErrorResponse(w, err, http.StatusServiceUnavailable)
}

// Модификация ответа
func modifyResponse(res *http.Response) error {
	res.Header.Set("X-Load-Balancer", "GoLB")
	return nil
}

func writeErrorResponse(w http.ResponseWriter, err error, code int) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errorResponce{
		Error: err.Error(),
		Code:  code,
	})
}
