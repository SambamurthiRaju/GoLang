package repo

import (
	"BankingAPI/internal/model"
	"BankingAPI/internal/storage"
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrInsufficient    = errors.New("insufficient funds")
	ErrAccountInactive = errors.New("account inactive")
)

type Repo struct {
	store *storage.InMemoryStore
	mu    sync.Mutex // to ensure atomic operations across multiple accounts
}

func (r *Repo) HandleFunc(s string, register func(w http.ResponseWriter, r *http.Request)) {
	panic("unimplemented")
}

func NewRepo(s *storage.InMemoryStore) *Repo {
	return &Repo{store: s}
}

func (r *Repo) CreateUser(ctx context.Context, u *model.User) (*model.User, error) {
	r.store.Mu.Lock()
	defer r.store.Mu.Unlock()
	if _, exists := r.store.EmailIndex[u.Email]; exists {
		return nil, errors.New("email already registered")
	}
	u.ID = uuid.NewString()
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	u.IsActive = true
	r.store.Users[u.ID] = u
	r.store.EmailIndex[u.Email] = u.ID
	return u, nil
}

func (r *Repo) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	r.store.Mu.RLock()
	defer r.store.Mu.RUnlock()
	id, ok := r.store.EmailIndex[email]
	if !ok {
		return nil, ErrNotFound
	}
	u := r.store.Users[id]
	return u, nil
}

func (r *Repo) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	r.store.Mu.RLock()
	defer r.store.Mu.RUnlock()
	u, ok := r.store.Users[id]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (r *Repo) CreateAccount(ctx context.Context, a *model.Account) (*model.Account, error) {
	r.store.Mu.Lock()
	defer r.store.Mu.Unlock()
	a.ID = uuid.NewString()
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
	if !a.IsActive {
		a.IsActive = true
	}
	r.store.Accounts[a.ID] = a
	return a, nil
}

func (r *Repo) GetAccount(ctx context.Context, id string) (*model.Account, error) {
	r.store.Mu.RLock()
	defer r.store.Mu.RUnlock()
	a, ok := r.store.Accounts[id]
	if !ok {
		return nil, ErrNotFound
	}
	return a, nil
}

func (r *Repo) ListAccountsByUser(ctx context.Context, userID string, currency string, minBalance *int64) ([]*model.Account, error) {
	r.store.Mu.RLock()
	defer r.store.Mu.RUnlock()
	out := []*model.Account{}
	for _, a := range r.store.Accounts {
		if a.UserID != userID || !a.IsActive {
			continue
		}
		if currency != "" && a.Currency != currency {
			continue
		}
		if minBalance != nil && a.Balance < *minBalance {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func (r *Repo) UpdateAccount(ctx context.Context, id string, name *string, isActive *bool) (*model.Account, error) {
	r.store.Mu.Lock()
	defer r.store.Mu.Unlock()
	a, ok := r.store.Accounts[id]
	if !ok {
		return nil, ErrNotFound
	}
	if name != nil {
		a.Name = *name
	}
	if isActive != nil {
		a.IsActive = *isActive
		if !a.IsActive {

		}
	}
	a.UpdatedAt = time.Now()
	return a, nil
}

// Soft delete: set IsActive = false
func (r *Repo) DeleteAccount(ctx context.Context, id string) error {
	r.store.Mu.Lock()
	defer r.store.Mu.Unlock()
	a, ok := r.store.Accounts[id]
	if !ok {
		return ErrNotFound
	}
	a.IsActive = false
	a.UpdatedAt = time.Now()
	return nil
}

func (r *Repo) Deposit(ctx context.Context, accountID string, amount int64, meta map[string]interface{}) (*model.Transaction, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	r.store.Mu.Lock()
	defer r.store.Mu.Unlock()
	a, ok := r.store.Accounts[accountID]
	if !ok {
		return nil, ErrNotFound
	}
	if !a.IsActive {
		return nil, ErrAccountInactive
	}
	a.Balance += amount
	a.UpdatedAt = time.Now()
	t := &model.Transaction{ID: uuid.NewString(), AccountID: a.ID, Type: model.Deposit, Amount: amount, Meta: meta, CreatedAt: time.Now()}
	r.store.Transactions[t.ID] = t
	return t, nil
}

func (r *Repo) Withdraw(ctx context.Context, accountID string, amount int64, meta map[string]interface{}) (*model.Transaction, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	r.store.Mu.Lock()
	defer r.store.Mu.Unlock()
	a, ok := r.store.Accounts[accountID]
	if !ok {
		return nil, ErrNotFound
	}
	if !a.IsActive {
		return nil, ErrAccountInactive
	}
	if a.Balance < amount {
		return nil, ErrInsufficient
	}
	a.Balance -= amount
	a.UpdatedAt = time.Now()
	t := &model.Transaction{ID: uuid.NewString(), AccountID: a.ID, Type: model.Withdraw, Amount: amount, Meta: meta, CreatedAt: time.Now()}
	r.store.Transactions[t.ID] = t
	return t, nil
}

func (r *Repo) Transfer(ctx context.Context, fromID, toID string, amount int64, meta map[string]interface{}) (*model.Transaction, *model.Transaction, error) {
	if amount <= 0 {
		return nil, nil, errors.New("amount must be positive")
	}
	// lock repo mutex to ensure atomic across accounts
	r.mu.Lock()
	defer r.mu.Unlock()

	r.store.Mu.Lock()
	defer r.store.Mu.Unlock()

	from, ok := r.store.Accounts[fromID]
	if !ok {
		return nil, nil, ErrNotFound
	}
	to, ok := r.store.Accounts[toID]
	if !ok {
		return nil, nil, ErrNotFound
	}
	if !from.IsActive || !to.IsActive {
		return nil, nil, ErrAccountInactive
	}
	if from.Balance < amount {
		return nil, nil, ErrInsufficient
	}

	from.Balance -= amount
	to.Balance += amount
	from.UpdatedAt = time.Now()
	to.UpdatedAt = time.Now()

	txnOut := &model.Transaction{ID: uuid.NewString(), AccountID: from.ID, Type: model.Withdraw, Amount: amount, Meta: meta, CreatedAt: time.Now()}
	r.store.Transactions[txnOut.ID] = txnOut
	txnIn := &model.Transaction{ID: uuid.NewString(), AccountID: to.ID, Type: model.Deposit, Amount: amount, Meta: meta, CreatedAt: time.Now()}
	r.store.Transactions[txnIn.ID] = txnIn
	return txnOut, txnIn, nil
}

// func (r *Repo) ListTransactions(ctx context.Context, userID string, accountID *string, typ *model.TransactionType, from, to *time.Time, limit, offset int) ([]*model.Transaction, error) {
// 	r.store.Mu.RLock()
// 	defer r.store.Mu.RUnlock()
// 	out := []*model.Transaction{}
// 	// build set of user's active accounts
// 	acctSet := map[string]struct{}{}
// 	for _, a := range r.store.Accounts {
// 		if a.UserID == userID {
// 			acctSet[a.ID] = struct{}{}
// 		}
// 	}
// 	for _, t := range r.store.Transactions {
// 		if _, ok := acctSet[t.AccountID]; !ok {
// 			continue
// 		}
// 		if accountID != nil && t.AccountID != *accountID {
// 			continue
// 		}
// 		if typ != nil && t.Type != *typ {
// 			continue
// 		}
// 		if from != nil && t.CreatedAt.Before(*from) {
// 			continue
// 		}
// 		if to != nil && t.CreatedAt.After(*to) {
// 			continue
// 		}
// 		out = append(out, t)
// 	}
// 	start := offset
// 	if start > len(out) {
// 		start = len(out)
// 	}
// 	end := start + limit
// 	if end > len(out) || limit == 0 {
// 		end = len(out)
// 	}
// 	return out[start:end], nil
// }
