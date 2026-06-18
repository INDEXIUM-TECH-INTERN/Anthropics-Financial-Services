package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/redis"

	goredis "github.com/redis/go-redis/v9"
)

const maxMemSessions = 1000

// simple in-memory fallback when Redis is not available
var (
	memSessions = make(map[string]*ChatSession)
	memMu       sync.RWMutex
)

type ChatSession struct {
	ID        string              `json:"id"`
	Title     string              `json:"title"`
	Messages  []messaging.Message `json:"messages"`
	UpdatedAt time.Time           `json:"updated_at"`
}

const sessionKeyPrefix = "chat:session:"

func sessionKey(id string) string {
	return sessionKeyPrefix + id
}

func listKey() string {
	return "chat:sessions:list"
}

// SaveSession saves or updates a chat session in Redis (with memory fallback).
func SaveSession(sess *ChatSession) error {
	if sess.ID == "" {
		return fmt.Errorf("session ID is required")
	}
	sess.UpdatedAt = time.Now()

	if redis.Client == nil {
		memMu.Lock()
		if len(memSessions) >= maxMemSessions {
			// Evict oldest session
			var oldestID string
			var oldestTime time.Time
			for id, s := range memSessions {
				if oldestID == "" || s.UpdatedAt.Before(oldestTime) {
					oldestID = id
					oldestTime = s.UpdatedAt
				}
			}
			delete(memSessions, oldestID)
		}
		memSessions[sess.ID] = sess
		memMu.Unlock()
		return nil
	}

	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(redis.Ctx, redis.OpTimeout)
	defer cancel()

	pipe := redis.Client.Pipeline()
	pipe.Set(ctx, sessionKey(sess.ID), data, 24*time.Hour)
	pipe.SAdd(ctx, listKey(), sess.ID)
	_, err = pipe.Exec(ctx)
	return err
}

// GetSession retrieves a session by ID.
func GetSession(id string) (*ChatSession, error) {
	if redis.Client == nil {
		memMu.RLock()
		sess, ok := memSessions[id]
		memMu.RUnlock()
		if ok {
			return sess, nil
		}
		return &ChatSession{
			ID:       id,
			Title:    "Cuộc trò chuyện mới",
			Messages: []messaging.Message{},
		}, nil
	}

	ctx, cancel := context.WithTimeout(redis.Ctx, redis.OpTimeout)
	defer cancel()

	data, err := redis.Client.Get(ctx, sessionKey(id)).Bytes()
	if err == goredis.Nil {
		return &ChatSession{
			ID:       id,
			Title:    "Cuộc trò chuyện mới",
			Messages: []messaging.Message{},
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var sess ChatSession
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// ListSessions returns all sessions (lightweight metadata).
func ListSessions() ([]*ChatSession, error) {
	if redis.Client == nil {
		memMu.RLock()
		defer memMu.RUnlock()
		sessions := []*ChatSession{}
		for _, sess := range memSessions {
			light := &ChatSession{
				ID:        sess.ID,
				Title:     sess.Title,
				UpdatedAt: sess.UpdatedAt,
				Messages:  nil,
			}
			sessions = append(sessions, light)
		}
		return sessions, nil
	}

	ctx, cancel := context.WithTimeout(redis.Ctx, redis.OpTimeout)
	defer cancel()

	ids, err := redis.Client.SMembers(ctx, listKey()).Result()
	if err != nil {
		return nil, err
	}

	sessions := []*ChatSession{}
	for _, id := range ids {
		sess, err := GetSession(id)
		if err != nil {
			continue
		}
		light := &ChatSession{
			ID:        sess.ID,
			Title:     sess.Title,
			UpdatedAt: sess.UpdatedAt,
			Messages:  nil,
		}
		sessions = append(sessions, light)
	}

	return sessions, nil
}

// DeleteSession removes a session.
func DeleteSession(id string) error {
	if redis.Client == nil {
		memMu.Lock()
		delete(memSessions, id)
		memMu.Unlock()
		return nil
	}
	ctx, cancel := context.WithTimeout(redis.Ctx, redis.OpTimeout)
	defer cancel()

	pipe := redis.Client.Pipeline()
	pipe.Del(ctx, sessionKey(id))
	pipe.SRem(ctx, listKey(), id)
	_, err := pipe.Exec(ctx)
	return err
}

