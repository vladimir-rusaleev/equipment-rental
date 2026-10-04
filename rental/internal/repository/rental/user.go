package rental

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"rent/internal/platform"
	"rent/rental/internal/entity"
)

func (m *memory) CreateUser(_ context.Context, u entity.User) (entity.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.emails[u.Email]; ok {
		return u, platform.ErrConflict
	}
	u.ID = platform.NewID()
	m.users[u.ID] = u
	m.emails[u.Email] = u.ID
	return u, nil
}

func (m *memory) UserByEmail(_ context.Context, email string) (entity.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[m.emails[email]]
	if !ok {
		return u, platform.ErrMissing
	}
	return u, nil
}

func (m *memory) User(_ context.Context, id string) (entity.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return u, platform.ErrMissing
	}
	return u, nil
}

func (m *memory) SetRole(_ context.Context, id, role string) (entity.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return u, platform.ErrMissing
	}
	if u.Role == entity.RoleAdmin && role != entity.RoleAdmin {
		n := 0
		for _, v := range m.users {
			if v.Role == entity.RoleAdmin {
				n++
			}
		}
		if n == 1 {
			return u, platform.ErrConflict
		}
	}
	u.Role = role
	m.users[id] = u
	return u, nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (m *memory) PutSession(_ context.Context, token, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[tokenHash(token)] = entity.StoredSession{UserID: id, ExpiresAt: time.Now().Add(24 * time.Hour)}
	return nil
}

func (m *memory) SessionUser(ctx context.Context, token string) (entity.User, error) {
	m.mu.RLock()
	s, ok := m.sessions[tokenHash(token)]
	m.mu.RUnlock()
	if !ok || time.Now().After(s.ExpiresAt) {
		return entity.User{}, platform.ErrUnauthenticated
	}
	return m.User(ctx, s.UserID)
}

func (m *memory) EnsureAdmin(_ context.Context, email, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.emails[email]; ok {
		if m.users[id].Role != entity.RoleAdmin {
			return platform.ErrConflict
		}
		return nil
	}
	user := entity.User{ID: platform.NewID(), Email: email, Role: entity.RoleAdmin, PasswordHash: hash}
	m.users[user.ID] = user
	m.emails[email] = user.ID
	return nil
}
