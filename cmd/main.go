package main

import (
	"LoadBalancer/internal/config"
	"LoadBalancer/internal/server"
	"LoadBalancer/internal/storage"
	"LoadBalancer/pkg/balancer"
	"LoadBalancer/pkg/rate_limiter"
	"context"
	"golang.org/x/sync/errgroup"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	//graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		cancel()
	}()

	cfg := config.LoadConfig("config/config.yaml")
	balancer := balancer.NewBalancer(ctx, cfg)
	limiter := rate_limiter.NewRateLimiter(cfg.Capacity, cfg.RefillRate)
	db := storage.NewPostgresRepo(ctx, cfg)
	clients, err := db.GetConfig(ctx)
	if err != nil {
		log.Printf("could not load custom config: %v\n", err)
	}
	for client, values := range clients {
		limiter.SetCustomLimit(client, values[0], values[1])
	}
	srv := server.NewServer(cfg, balancer, limiter)

	errG, gCtx := errgroup.WithContext(ctx)

	errG.Go(func() error {
		log.Printf("starting server on port: %s", cfg.ServerPort)
		return srv.Start()
	})

	errG.Go(func() error {
		<-gCtx.Done()
		log.Println("shutting down server...")
		return srv.Stop(gCtx)
	})
	errG.Go(func() error {
		<-gCtx.Done()
		log.Println("clearing limiter...")
		limiter.Clear()
		return nil
	})
	errG.Go(func() error {
		<-gCtx.Done()
		log.Println("closing database...")
		db.Close()
		return nil
	})
	if err = errG.Wait(); err != nil {
		log.Printf("exit reason: %s \n", err)
	}
	log.Println("app shutdown")
}
