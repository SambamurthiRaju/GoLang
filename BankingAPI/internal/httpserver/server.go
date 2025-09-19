package httpservers

import (
	"BankingAPI/internal/auth"
	"BankingAPI/internal/middleware"
	"BankingAPI/internal/model"
	"BankingAPI/internal/repo"
	"BankingAPI/internal/storage"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Server ties repo and router
type Server struct {
	repo   *repo.Repo
	router *mux.Router
}

// NewServer builds router, repo and handlers
func NewServer() *Server {
	store := storage.NewInMemoryStore()
	r := repo.NewRepo(store)
	s := &Server{repo: r}
	mx := mux.NewRouter()
	// global recover middleware
	mx.Use(middleware.Recoverer)

	// auth handlers
	authH := &auth.AuthHandler{Repo: r}
	mx.HandleFunc("/auth/register", authH.Register).Methods("POST")
	mx.HandleFunc("/auth/login", authH.Login).Methods("POST")

	// protected routes
	pr := mx.PathPrefix("/").Subrouter()
	pr.Use(middleware.Auth)
	pr.HandleFunc("/auth/me", authH.Me).Methods("GET")

	// accounts
	pr.HandleFunc("/accounts", s.createAccount).Methods("POST")
	pr.HandleFunc("/accounts", s.listAccounts).Methods("GET")
	pr.HandleFunc("/accounts/{id}", s.getAccount).Methods("GET")
	pr.HandleFunc("/accounts/{id}", s.updateAccount).Methods("PUT")
	pr.HandleFunc("/accounts/{id}", s.deleteAccount).Methods("DELETE")
	pr.HandleFunc("/accounts/{id}/deposit", s.deposit).Methods("POST")
	pr.HandleFunc("/accounts/{id}/withdraw", s.withdraw).Methods("POST")

	// transfers
	pr.HandleFunc("/transfers", s.transfer).Methods("POST")

	// transactions listing
	// pr.HandleFunc("/accounts/transactions", s.listTransactions).Methods("GET")

	s.router = mx
	s.repo = r
	return s
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) Shutdown(ctx context.Context) error {
	_ = ctx
	return nil
}

func getUserID(r *http.Request) string { return r.Context().Value("user_id").(string) }

// createAccount
type createAccountReq struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

// @Summary Create account
// @Tags accounts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body createAccountReq true "create account"
// @Success 201 {object} model.Account
// @Failure 400 {string} string
// @Router /accounts [post]
func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var req createAccountReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Name == "" || req.Currency == "" {
		http.Error(w, "name and currency required", http.StatusBadRequest)
		return
	}
	userID := getUserID(r)
	acc := &model.Account{
		UserID:    userID,
		Name:      req.Name,
		Currency:  req.Currency,
		Balance:   0,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	created, err := s.repo.CreateAccount(r.Context(), acc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// @Summary List accounts
// @Tags accounts
// @Security BearerAuth
// @Param currency query string false "currency filter"
// @Param min_balance query number false "min balance (minor units)"
// @Produce json
// @Success 200 {array} model.Account
// @Router /accounts [get]
func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	currency := q.Get("currency")
	var min *int64
	if v := q.Get("min_balance"); v != "" {
		if v2, err := strconv.ParseInt(v, 10, 64); err == nil {
			min = &v2
		}
	}
	userID := getUserID(r)
	list, _ := s.repo.ListAccountsByUser(r.Context(), userID, currency, min)
	json.NewEncoder(w).Encode(list)
}

// @Summary Get account
// @Tags accounts
// @Security BearerAuth
// @Param id path string true "account id"
// @Produce json
// @Success 200 {object} model.Account
// @Failure 404 {string} string
// @Router /accounts/{id} [get]
func (s *Server) getAccount(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	a, err := s.repo.GetAccount(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if a.UserID != getUserID(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	json.NewEncoder(w).Encode(a)
}

type updateAccountReq struct {
	Name     *string `json:"name,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// @Summary Update account
// @Tags accounts
// @Security BearerAuth
// @Param id path string true "account id"
// @Param body body updateAccountReq true "update"
// @Produce json
// @Success 200 {object} model.Account
// @Router /accounts/{id} [put]
func (s *Server) updateAccount(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	acc, err := s.repo.GetAccount(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if acc.UserID != getUserID(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req updateAccountReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	updated, err := s.repo.UpdateAccount(r.Context(), id, req.Name, req.IsActive)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(updated)
}

// @Summary Soft delete account
// @Tags accounts
// @Security BearerAuth
// @Param id path string true "account id"
// @Success 204 {string} string
// @Router /accounts/{id} [delete]
func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	a, err := s.repo.GetAccount(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if a.UserID != getUserID(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	_ = s.repo.DeleteAccount(r.Context(), id)
	w.WriteHeader(http.StatusNoContent)
}

type amountReq struct {
	Amount int64                  `json:"amount"`
	Meta   map[string]interface{} `json:"meta,omitempty"`
}

// @Summary Deposit
// @Tags accounts
// @Security BearerAuth
// @Param id path string true "account id"
// @Param body body amountReq true "deposit amount"
// @Produce json
// @Success 200 {object} model.Transaction
// @Router /accounts/{id}/deposit [post]
func (s *Server) deposit(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	acc, err := s.repo.GetAccount(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if acc.UserID != getUserID(r) || !acc.IsActive {
		http.Error(w, "forbidden or inactive", http.StatusForbidden)
		return
	}
	var req amountReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	t, err := s.repo.Deposit(r.Context(), id, req.Amount, req.Meta)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(t)
}

// @Summary Withdraw
// @Tags accounts
// @Security BearerAuth
// @Param id path string true "account id"
// @Param body body amountReq true "withdraw amount"
// @Produce json
// @Success 200 {object} model.Transaction
// @Router /accounts/{id}/withdraw [post]
func (s *Server) withdraw(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	acc, err := s.repo.GetAccount(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if acc.UserID != getUserID(r) || !acc.IsActive {
		http.Error(w, "forbidden or inactive", http.StatusForbidden)
		return
	}
	var req amountReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	t, err := s.repo.Withdraw(r.Context(), id, req.Amount, req.Meta)
	if err != nil {
		if err == repo.ErrInsufficient {
			http.Error(w, "insufficient funds", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(t)
}

type transferReq struct {
	FromAccountID string                 `json:"from_account_id"`
	ToAccountID   string                 `json:"to_account_id"`
	Amount        int64                  `json:"amount"`
	Meta          map[string]interface{} `json:"meta,omitempty"`
}

// @Summary Transfer
// @Tags transfers
// @Security BearerAuth
// @Accept json
// @Param body body transferReq true "transfer"
// @Produce json
// @Success 200 {object} map[string]model.Transaction
// @Router /transfers [post]
func (s *Server) transfer(w http.ResponseWriter, r *http.Request) {
	var req transferReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	from, err := s.repo.GetAccount(r.Context(), req.FromAccountID)
	if err != nil || from.UserID != getUserID(r) {
		http.Error(w, "invalid from account", http.StatusBadRequest)
		return
	}
	if !from.IsActive {
		http.Error(w, "from account inactive", http.StatusBadRequest)
		return
	}
	to, err := s.repo.GetAccount(r.Context(), req.ToAccountID)
	if err != nil {
		http.Error(w, "to account not found", http.StatusBadRequest)
		return
	}
	if !to.IsActive {
		http.Error(w, "to account inactive", http.StatusBadRequest)
		return
	}
	out, in, err := s.repo.Transfer(r.Context(), req.FromAccountID, req.ToAccountID, req.Amount, req.Meta)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(map[string]*model.Transaction{
		"withdraw_txn": out,
		"deposit_txn":  in,
	})
}

// // @Summary List transactions
// // @Tags transactions
// // @Security BearerAuth
// // @Param account_id query string false "account id"
// // @Param type query string false "DEPOSIT|WITHDRAW"
// // @Param from query string false "from date RFC3339"
// // @Param to query string false "to date RFC3339"
// // @Param limit query int false "limit"
// // @Param offset query int false "offset"
// // @Produce json
// // @Success 200 {array} model.Transaction
// // @Router /accounts/transactions [get]
// func (s *Server) listTransactions(w http.ResponseWriter, r *http.Request) {
// 	q := r.URL.Query()
// 	var accountID *string
// 	if v := q.Get("account_id"); v != "" {
// 		accountID = &v
// 	}
// 	var typ *model.TransactionType
// 	if v := q.Get("type"); v != "" {
// 		t := model.TransactionType(v)
// 		typ = &t
// 	}
// 	var from, to *time.Time
// 	if v := q.Get("from"); v != "" {
// 		if t, err := time.Parse(time.RFC3339, v); err == nil {
// 			from = &t
// 		}
// 	}
// 	if v := q.Get("to"); v != "" {
// 		if t, err := time.Parse(time.RFC3339, v); err == nil {
// 			to = &t
// 		}
// 	}
// 	limit := 100
// 	offset := 0
// 	if v := q.Get("limit"); v != "" {
// 		if p, err := strconv.Atoi(v); err == nil {
// 			limit = p
// 		}
// 	}
// 	if v := q.Get("offset"); v != "" {
// 		if p, err := strconv.Atoi(v); err == nil {
// 			offset = p
// 		}
// 	}
// 	list, err := s.repo.ListTransactions(r.Context(), getUserID(r), accountID, typ, from, to, limit, offset)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	json.NewEncoder(w).Encode(list)
// }
