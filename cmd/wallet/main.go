package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/joho/godotenv"

    "wallet/internal/httpapi"
    "wallet/internal/store"
)


func main() {
    _ = godotenv.Load("config.env")

    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        log.Fatal("DATABASE_URL is required")
    }

    pool, err := store.NewPool(context.Background(), dbURL)
    if err != nil {
        log.Fatalf("db connect: %v", err)
    }
    defer pool.Close()

    repo := store.NewRepository(pool)
    if err := repo.EnsureSchema(context.Background()); err != nil {
        log.Fatalf("ensure schema: %v", err)
    }
    api := httpapi.NewServer(repo)

    r := chi.NewRouter()
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    // Регистрируем HTTP-маршруты API
    api.MountRoutes(r)

    srv := &http.Server{ 
        Addr: ":8080",
        Handler: r,
        ReadHeaderTimeout: 10 * time.Second,
    }

    go func() {
        log.Printf("listening on %s", srv.Addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("listen: %v", err)
        }
    }()

    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    <-stop

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    _ = srv.Shutdown(ctx)
}


