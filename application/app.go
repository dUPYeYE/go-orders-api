package application

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type App struct {
  router    http.Handler
  rdb       *redis.Client
  config    Config
}

func New(config Config) *App {
  app := &App{
    rdb:    redis.NewClient(&redis.Options{
      Addr: config.RedisAddr,
    }),
    config: config,
  }

  app.loadRoutes()

  return app
}

func (a *App) Start(ctx context.Context) error {
  server := &http.Server{
    Addr:   fmt.Sprintf(":%d", a.config.ServerPort),
    Handler: a.router,
  }

  err := a.rdb.Ping(ctx).Err()
  if err != nil {
    return fmt.Errorf("failed to connect to redis: %w", err)
  }

  defer func() {
    if err := a.rdb.Close(); err != nil {
      fmt.Println("failed to close redis connection", err)
    }
  }()

  fmt.Println("starting server")

  ch := make(chan error, 1)

  go func() {
    err = server.ListenAndServe()
    if err != nil {
      ch <- fmt.Errorf("failed to listen to server: %w", err)
    }
    close(ch)
  }()

  select {
  case err = <-ch:
    return err
  case <-ctx.Done():
    timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err = server.Shutdown(timeout)
  }


  return nil
}
