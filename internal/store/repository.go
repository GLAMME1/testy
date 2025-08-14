package store

import (
    "context"
    "encoding/binary"
    "errors"
    "fmt"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

var ErrInsufficientFunds = errors.New("insufficient funds")

type OperationType string

const (
    OperationDeposit  OperationType = "DEPOSIT"
    OperationWithdraw OperationType = "WITHDRAW"
)

type Repository struct {
    pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) EnsureSchema(ctx context.Context) error {
    _, err := r.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY,
    balance BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`)
    return err
}

// ChangeBalance applies deposit/withdraw atomically with serialization per wallet.
func (r *Repository) ChangeBalance(ctx context.Context, walletID uuid.UUID, op OperationType, amount int64) (int64, error) {
    // Преобразуем операцию в приращение баланса
    var delta int64
    switch op {
    case OperationDeposit:
        delta = amount
    case OperationWithdraw:
        delta = -amount
    default:
        return 0, fmt.Errorf("unknown operation: %s", op)
    }

    tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return 0, err
    }
    defer func() { _ = tx.Rollback(ctx) }()

    // Последовательный доступ к одному кошельку через advisory lock
    var k1, k2 int32
    b := walletID
    k1 = int32(binary.BigEndian.Uint32(b[0:4]))
    k2 = int32(binary.BigEndian.Uint32(b[4:8]))
    if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, $2)`, k1, k2); err != nil {
        return 0, err
    }

    // Обеспечиваем наличие записи
    if _, err = tx.Exec(ctx, `INSERT INTO wallets (id, balance) VALUES ($1, 0) ON CONFLICT (id) DO NOTHING`, walletID); err != nil {
        return 0, err
    }

    // Атомарное обновление с проверкой на отрицательный баланс
    var newBalance int64
    err = tx.QueryRow(ctx, `
        UPDATE wallets
        SET balance = balance + $2, updated_at = now()
        WHERE id = $1 AND (balance + $2) >= 0
        RETURNING balance
    `, walletID, delta).Scan(&newBalance)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return 0, ErrInsufficientFunds
        }
        return 0, err
    }

    if err := tx.Commit(ctx); err != nil {
        return 0, err
    }
    return newBalance, nil
}

func (r *Repository) GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
    var balance int64
    err := r.pool.QueryRow(ctx, `SELECT balance FROM wallets WHERE id=$1`, walletID).Scan(&balance)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return 0, nil
        }
        return 0, err
    }
    return balance, nil
}


