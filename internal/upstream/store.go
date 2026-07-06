package upstream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/pkg/crypto"
)

// ErrProviderPaused is returned when routing is attempted against a paused provider.
var ErrProviderPaused = errors.New("provider is paused")

// Store selects provider connections and persists lock state.
type Store struct {
	DB            *db.DB
	EncryptionKey []byte
}

func connectionAAD(id string) []byte {
	return []byte("provider_connection:" + id)
}

// DecryptKey returns the plaintext API key for a connection.
func (s *Store) DecryptKey(conn *db.ProviderConnection) (string, error) {
	if conn.APIKeyEncrypted == nil {
		return "", nil
	}
	key, err := crypto.DecryptString(*conn.APIKeyEncrypted, s.EncryptionKey, connectionAAD(conn.ID))
	if err != nil {
		return "", fmt.Errorf("decrypt connection %s key: %w", conn.ID, err)
	}
	return key, nil
}

// Select picks the next available connection for a provider/upstream model.
func (s *Store) Select(ctx context.Context, providerID, upstreamModel, strategy string, stickyLimit int, exclude map[string]struct{}) (*db.ProviderConnection, error) {
	prov, err := s.DB.GetProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if prov.Status == "paused" {
		return nil, ErrProviderPaused
	}

	conns, err := s.DB.ListProviderConnections(ctx, providerID, true)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var available []db.ProviderConnection
	for _, c := range conns {
		if exclude != nil {
			if _, ok := exclude[c.ID]; ok {
				continue
			}
		}
		if c.LockedUntil != nil {
			if t, err := time.Parse(time.RFC3339, *c.LockedUntil); err == nil && t.After(now) {
				continue
			}
		}
		if IsModelLockActive(c.ModelLocks, upstreamModel, now) {
			continue
		}
		available = append(available, c)
	}
	if len(available) == 0 {
		return nil, db.ErrNotFound
	}

	if strategy == "round-robin" && stickyLimit < 1 {
		stickyLimit = 1
	}
	if strategy != "round-robin" {
		return &available[0], nil
	}

	// Round-robin: pick least recently used; honor sticky count on most recent.
	var lru *db.ProviderConnection
	for i := range available {
		c := &available[i]
		if lru == nil {
			lru = c
			continue
		}
		if c.LastUsedAt == nil {
			lru = c
			continue
		}
		if lru.LastUsedAt == nil {
			continue
		}
		tC, _ := time.Parse(time.RFC3339, *c.LastUsedAt)
		tL, _ := time.Parse(time.RFC3339, *lru.LastUsedAt)
		if tC.Before(tL) {
			lru = c
		}
	}
	if lru == nil {
		return &available[0], nil
	}

	useCount := lru.ConsecutiveUseCount + 1
	if lru.LastUsedAt != nil && useCount <= stickyLimit {
		nowS := now.Format(time.RFC3339)
		updated, _ := s.DB.UpdateProviderConnection(ctx, lru.ID, db.UpdateProviderConnectionParams{
			LastUsedAt: &nowS, ConsecutiveUseCount: &useCount,
		})
		if updated != nil {
			return updated, nil
		}
		return lru, nil
	}

	// Rotate to next by priority after current LRU holder.
	for i := range available {
		if available[i].ID == lru.ID {
			next := &available[(i+1)%len(available)]
			one := 1
			nowS := now.Format(time.RFC3339)
			updated, _ := s.DB.UpdateProviderConnection(ctx, next.ID, db.UpdateProviderConnectionParams{
				LastUsedAt: &nowS, ConsecutiveUseCount: &one,
			})
			if updated != nil {
				return updated, nil
			}
			return next, nil
		}
	}
	return lru, nil
}

// MarkUnavailable locks a connection after upstream failure.
func (s *Store) MarkUnavailable(ctx context.Context, connID, upstreamModel string, status int, errText string, backoffLevel int, retryAfter time.Duration) error {
	class := ClassifyError(status, errText, backoffLevel)
	if !class.ShouldRotate {
		return nil
	}
	cooldown := class.Cooldown
	if retryAfter > 0 {
		cooldown = retryAfter
	}
	until := time.Now().UTC().Add(cooldown)
	locks := map[string]string{}
	if conn, err := s.DB.GetProviderConnection(ctx, connID); err == nil {
		locks = conn.ModelLocks
		if locks == nil {
			locks = map[string]string{}
		}
	}
	key := upstreamModel
	if key == "" {
		key = "__all__"
	}
	locks[key] = until.Format(time.RFC3339)
	unavailable := "unavailable"
	code := status
	errMsg := errText
	if len(errMsg) > 200 {
		errMsg = errMsg[:200]
	}
	at := until.Format(time.RFC3339)
	bl := class.NewBackoffLevel
	if bl == 0 {
		bl = backoffLevel
	}
	_, err := s.DB.UpdateProviderConnection(ctx, connID, db.UpdateProviderConnectionParams{
		ModelLocks: locks, TestStatus: &unavailable, LastError: &errMsg,
		ErrorCode: &code, LastErrorAt: &at, BackoffLevel: &bl,
	})
	return err
}

// ClearSuccess clears model lock and error state after a successful call.
func (s *Store) ClearSuccess(ctx context.Context, connID, upstreamModel string) error {
	conn, err := s.DB.GetProviderConnection(ctx, connID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	locks := map[string]string{}
	for k, v := range conn.ModelLocks {
		if upstreamModel != "" && k == upstreamModel {
			continue
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil && t.After(now) {
			locks[k] = v
		}
	}
	active := "active"
	empty := ""
	zero := 0
	_, err = s.DB.UpdateProviderConnection(ctx, connID, db.UpdateProviderConnectionParams{
		ModelLocks: locks, TestStatus: &active, LastError: &empty,
		ClearErrorCode: true, ClearLastErrorAt: true, BackoffLevel: &zero,
	})
	return err
}