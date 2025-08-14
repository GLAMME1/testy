//go:build integration
// +build integration

package store

import (
    "context"
    "sync"
    "testing"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgxpool"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
    t.Helper()
    dbURL := "postgres://wallet:wallet@localhost:5432/wallet?sslmode=disable"
    pool, err := NewPool(context.Background(), dbURL)
    if err != nil {
        t.Fatalf("db: %v", err)
    }
    return pool
}

func TestChangeBalance_Sequential(t *testing.T) {
    pool := newTestPool(t)
    defer pool.Close()
    repo := NewRepository(pool)
    ctx := context.Background()
    if err := repo.EnsureSchema(ctx); err != nil {
        t.Fatal(err)
    }
    id := uuid.New()
    // Ensure zero balance
    if _, err := repo.ChangeBalance(ctx, id, OperationDeposit, 0); err != nil {
        t.Fatal(err)
    }
    if b, err := repo.GetBalance(ctx, id); err != nil || b != 0 {
        t.Fatalf("expected 0, got %d err=%v", b, err)
    }

    b, err := repo.ChangeBalance(ctx, id, OperationDeposit, 100)
    if err != nil || b != 100 {
        t.Fatalf("expected 100, got %d err=%v", b, err)
    }
    b, err = repo.ChangeBalance(ctx, id, OperationWithdraw, 40)
    if err != nil || b != 60 {
        t.Fatalf("expected 60, got %d err=%v", b, err)
    }
}

func TestChangeBalance_Concurrent(t *testing.T) {
    pool := newTestPool(t)
    defer pool.Close()
    repo := NewRepository(pool)
    ctx := context.Background()
    if err := repo.EnsureSchema(ctx); err != nil {
        t.Fatal(err)
    }
    id := uuid.New()
    const workers = 200
    const amount = int64(5)

    var wg sync.WaitGroup
    wg.Add(workers)
    for i := 0; i < workers; i++ {
        go func() {
            defer wg.Done()
            if _, err := repo.ChangeBalance(ctx, id, OperationDeposit, amount); err != nil {
                t.Errorf("op failed: %v", err)
            }
        }()
    }
    wg.Wait()

    got, err := repo.GetBalance(ctx, id)
    if err != nil {
        t.Fatal(err)
    }
    want := int64(workers) * amount
    if got != want {
        t.Fatalf("balance mismatch: got=%d want=%d", got, want)
    }
}


