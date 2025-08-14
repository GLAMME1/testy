package httpapi

import (
    "context"
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"

    "wallet/internal/store"
)

//  минимальная зависимость HTTP-слоя для работы с кошельками
type WalletService interface {
    ChangeBalance(rctx context.Context, walletID uuid.UUID, op store.OperationType, amount int64) (int64, error)
    GetBalance(rctx context.Context, walletID uuid.UUID) (int64, error)
}

type Server struct {
    repo WalletService
}

// создаёт HTTP-сервер с зависимостями
func NewServer(repo WalletService) *Server { return &Server{repo: repo} }

type changeRequest struct {
    WalletID      uuid.UUID            `json:"valletId"`
    OperationType store.OperationType  `json:"operationType"`
    Amount        int64                `json:"amount"`
}

type balanceResponse struct {
    WalletID uuid.UUID `json:"walletId"`
    Balance  int64     `json:"balance"`
}

//  регистрирует маршруты API на роутере.
func (s *Server) MountRoutes(r chi.Router) {
    r.Route("/api/v1", func(r chi.Router) {
        r.Post("/wallet", s.handleChange)
        r.Get("/wallets/{id}", s.handleGet)
    })
}

// применяет операцию к кошельку и возвращает новый баланс
func (s *Server) handleChange(w http.ResponseWriter, r *http.Request) {
    var req changeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid json", http.StatusBadRequest)
        return
    }

    if req.Amount <= 0 {
        http.Error(w, "amount must be positive", http.StatusBadRequest)
        return
    }

    newBalance, err := s.repo.ChangeBalance(r.Context(), req.WalletID, req.OperationType, req.Amount)
    if err != nil {
        if err == store.ErrInsufficientFunds {
            http.Error(w, err.Error(), http.StatusUnprocessableEntity)
            return
        }
        http.Error(w, "server error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(balanceResponse{WalletID: req.WalletID, Balance: newBalance})
}

// возвращает текущий баланс кошелька по его идентификатору
func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    walletID, err := uuid.Parse(idStr)
    if err != nil {
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }

    balance, err := s.repo.GetBalance(r.Context(), walletID)
    if err != nil {
        http.Error(w, "server error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(balanceResponse{WalletID: walletID, Balance: balance})
}



