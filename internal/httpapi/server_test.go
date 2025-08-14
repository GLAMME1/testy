package httpapi

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"

    "wallet/internal/store"
)

func TestHandleChange_Valid(t *testing.T) {
    // Minimal fake via closure-backed implementation
    type fake struct{}
    var _ WalletService = (*fake)(nil)
    func (fake) ChangeBalance(_ context.Context, _ uuid.UUID, _ store.OperationType, _ int64) (int64, error) { return 10, nil }
    func (fake) GetBalance(_ context.Context, _ uuid.UUID) (int64, error) { return 10, nil }
    f := &fake{}
    s := NewServer(f)

    r := chi.NewRouter()
    s.MountRoutes(r)

    id := uuid.New()
    body := changeRequest{WalletID: id, OperationType: store.OperationDeposit, Amount: 10}
    b, _ := json.Marshal(body)
    req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(b))
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    if w.Code == 500 || w.Code == 400 {
        t.Fatalf("unexpected status: %d", w.Code)
    }
}


