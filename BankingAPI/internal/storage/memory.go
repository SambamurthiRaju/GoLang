package storage

import (
	"BankingAPI/internal/model"
	"sync"
)

// InMemoryStore is a thread-safe in-memory store implementation.
type InMemoryStore struct {
	Mu           sync.RWMutex
	Users        map[string]*model.User
	Accounts     map[string]*model.Account
	Transactions map[string]*model.Transaction
	EmailIndex   map[string]string // email -> userID
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		Users:        make(map[string]*model.User),
		Accounts:     make(map[string]*model.Account),
		Transactions: make(map[string]*model.Transaction),
		EmailIndex:   make(map[string]string),
	}
}
