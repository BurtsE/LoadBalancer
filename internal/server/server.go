package server

import (
	"LoadBalancer/internal/config"
	"LoadBalancer/pkg/balancer"
	"LoadBalancer/pkg/metrics"
	"context"
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type errorResponse struct {
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
	metrics  *metrics.Metrics
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
		Handler: http.DefaultServeMux,
	}
	reg := prometheus.NewRegistry()
	m := metrics.NewMetrics(reg)
	s.metrics = m

	http.Handle("/metrics/", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	http.Handle("/ping/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("pong")) }))
	http.Handle("/", http.HandlerFunc(s.handleRequest))
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
	defer s.writeMetric(r, time.Now())

	clientIP := r.RemoteAddr
	if !s.limiter.Allow(clientIP) {
		writeErrorResponse(w, fmt.Errorf("rate limit exceeded"), http.StatusTooManyRequests)
		return
	}
	s.proxy.ServeHTTP(w, r)
}

func (s *Server) writeMetric(r *http.Request, start time.Time) {
	s.metrics.Requests.WithLabelValues(r.Method, r.RequestURI).Inc()
	s.metrics.Duration.WithLabelValues().Observe(time.Since(start).Seconds())

}

// Настройка прокси
func (s *Server) director(req *http.Request) {
	backend := s.balancer.GetNextURL()
	if backend == nil {
		log.Println("no backend available")
		return
	}

	log.Printf("Forwarding request to %s | %s %s", backend.Host, req.Method, req.URL.Path)
	req.URL.Scheme = backend.Scheme
	req.URL.Host = backend.Host
	req.Header.Set("X-Forwarded-For", req.RemoteAddr)

	ctx := context.WithValue(req.Context(), "targetUrl", backend.Host)
	*req = *req.WithContext(ctx)
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
	json.NewEncoder(w).Encode(errorResponse{
		Error: err.Error(),
		Code:  code,
	})
}
