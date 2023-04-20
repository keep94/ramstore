// Copyright 2013 Travis Keep. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or
// at http://opensource.org/licenses/BSD-3-Clause.

// Package ramstore implements an in-memory session store for Gorilla Web Toolkit.
// In-memory sessions expire after a set time of inactivity.
package ramstore

import (
	"encoding/base32"
	"github.com/keep94/securecookie"
	"github.com/keep94/sessions"
	"net/http"
	"strings"
)

// Clients may choose to write their own implementation of the SessionData
// interface instead of using RAMSessions.
type SessionData interface {
	// GetData gets session data by id. GetData must return a shallow copy of
	// the data.
	GetData(id string) (map[interface{}]interface{}, error)
	// SaveData saves session data by id. SaveData must make a shallow copy of
	// values before saving.
	SaveData(id string, values map[interface{}]interface{}) error
}

// RAMStore is an in-memory session store for Gorilla Web Toolkit. This store
// makes shallow copies of maps, so value objects such as string and int can be
// safely used with in-memory sessions with no regard for synchronization.
// Care must be taken with reference types such as pointers, maps, or slices.
// To what these reference refer must be treated as frozen to prevent
// contention with other go-routines.
type RAMStore struct {
	Options *sessions.Options
	Data    *RAMSessions
	// Client sets either Data or SData leaving the other nil. If both Data and
	// SData are non-nil then SData takes precedence.
	SData SessionData
}

// NewRAMStore creates a new in-memory session store. maxAge is the maximum
// time of inactivity in seconds before a session expires. NewRamStore is
// a convenience routine that returns a *RamStore with the Data field set
// and the SData field nil. The returned *RamStore uses '/' as the cookie
// path.
func NewRAMStore(maxAge int) *RAMStore {
	return &RAMStore{
		Options: &sessions.Options{
			Path: "/"},
		Data: NewRAMSessions(maxAge)}
}

// Get retrieves the session. name is the name of the cookie storing the
// session ID. If Get is called a second time with the same request pointer,
// the session is retrieved from the request's context rather than from this
// store. Callers should call context.Clear() in a defer statement after
// calling Get.
func (s *RAMStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

// New fetches the session from this store. name is the name of the cookie
// holding the session ID. Get calls New if the session is not already cached
// in the request's context. This implementation of New never returns a non-nil
// error if client is storing sessions data in a *RamSessions instance.
func (s *RAMStore) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(s, name)
	defaultOptions := *s.Options
	session.Options = &defaultOptions
	session.IsNew = true
	if c, errCookie := r.Cookie(name); errCookie == nil {
		session.ID = c.Value
		if err := s.load(session); err != nil {
			return session, err
		}
	}
	return session, nil
}

// Save saves a session to the store. If the session has no ID, Save assigns
// a random one. This implementation of Save never returns a non-nil error
// if client is storing sessions data in a *RamSessions instance.
func (s *RAMStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	if session.ID == "" {
		session.ID = strings.TrimRight(
			base32.StdEncoding.EncodeToString(
				securecookie.GenerateRandomKey(32)), "=")
	}
	if err := s.save(session); err != nil {
		return err
	}
	http.SetCookie(w, sessions.NewCookie(session.Name(), session.ID, session.Options))
	return nil
}

func (s *RAMStore) save(session *sessions.Session) error {
	return s.getData().SaveData(session.ID, session.Values)
}

func (s *RAMStore) load(session *sessions.Session) error {
	sessionData, err := s.getData().GetData(session.ID)
	if err != nil {
		return err
	}
	if sessionData != nil {
		session.Values = sessionData
		session.IsNew = false
	}
	return nil
}

func (s *RAMStore) getData() SessionData {
	if s.SData != nil {
		return s.SData
	}
	return s.Data
}
