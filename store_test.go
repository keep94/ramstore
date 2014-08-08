// Copyright 2013 Travis Keep. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or
// at http://opensource.org/licenses/BSD-3-Clause.

package ramstore

import (
  "errors"
  "net/http"
  "testing"
)

var (
  errSessionData = errors.New("ramstore_test: Error in session data.")
)

func TestChangeOptions(t *testing.T) {
  request := &http.Request{}
  s := NewRAMStore(900)
  session, err := s.Get(request, "session-cookie")
  if err != nil {
    t.Errorf("Expected no error, got %v", err)
  }
  session.Options.MaxAge = 12345
  anotherRequest := &http.Request{}
  session, err = s.Get(anotherRequest, "session-cookie")
  if err != nil {
    t.Errorf("Expected no error, got %v", err)
  }
  if session.Options.MaxAge == 12345 {
    t.Error("Setting options in session should not override default options in store.")
  }
}

func TestGetNew(t *testing.T) {
  request := &http.Request{}
  s := NewRAMStore(900)
  session, err := s.Get(request, "session-cookie")
  if err != nil {
    t.Errorf("Expected no error, got %v", err)
  }
  if !session.IsNew {
    t.Error("Expected session to be new.")
  }
  if session.ID != "" {
    t.Errorf("Expected empty session ID, got %v", session.ID)
  }
  if len(session.Values) != 0 {
    t.Errorf("Expected empty session.Values, got %v", session.Values)
  }
}

func TestSaveGet(t *testing.T) {
  // Get new session
  request := &http.Request{}
  s := NewRAMStore(900)
  session, err := s.Get(request, "session-cookie")
  if err != nil {
    t.Errorf("Expected no error getting session, got %v", err)
  }
  // Populate session with data and save session
  session.Values["count"] = 3
  w := &responseWriter{http.Header{}}
  err = session.Save(request, w)
  if err != nil {
    t.Errorf("Expected no error saving, got %v", err)
  }

  // Get saved session
  anotherRequest := &http.Request{Header: http.Header{"Cookie": w.Header()["Set-Cookie"]}}
  session, err = s.Get(anotherRequest, "session-cookie")
  if err != nil {
    t.Errorf("Expected no error getting session, got %v", err)
  }
  if session.IsNew {
    t.Error("Expected session not to be new.")
  }
  if output := len(session.Values); output != 1 {
    t.Errorf("Expected session.Values to be of length 1, got %v", output)
  }
  if output := session.Values["count"]; output != 3 {
    t.Errorf("Expected 3, got %v", output)
  }

  // Now simulate an expired session by using a new store with another request
  // that has the same cookie
  thirdRequest := &http.Request{Header: http.Header{"Cookie": w.Header()["Set-Cookie"]}}
  s = NewRAMStore(900)
  session, err = s.Get(thirdRequest, "session-cookie")
  if err != nil {
    t.Errorf("Expected no error getting session, got %v", err)
  }
  if !session.IsNew {
    t.Error("Expected session to be new.")
  }
  if session.ID == "" {
    t.Error("Expected non-empty session ID")
  }
  if len(session.Values) != 0 {
    t.Errorf("Expected empty session.Values, got %v", session.Values)
  }
}

func TestUseSData(t *testing.T) {
  // Get new session
  request := &http.Request{}
  s := NewRAMStore(900)
  s = withSData(s, s.Data)
  session, err := s.Get(request, "session-cookie")
  if err != nil {
    t.Errorf("Expected no error getting session, got %v", err)
  }
  // Populate session with data and save session
  session.Values["count"] = 3
  w := &responseWriter{http.Header{}}
  err = session.Save(request, w)
  if err != nil {
    t.Errorf("Expected no error saving, got %v", err)
  }

  // Get saved session
  anotherRequest := &http.Request{Header: http.Header{"Cookie": w.Header()["Set-Cookie"]}}
  session, err = s.Get(anotherRequest, "session-cookie")
  if err != nil {
    t.Errorf("Expected no error getting session, got %v", err)
  }
  if session.IsNew {
    t.Error("Expected session not to be new.")
  }
  if output := len(session.Values); output != 1 {
    t.Errorf("Expected session.Values to be of length 1, got %v", output)
  }
  if output := session.Values["count"]; output != 3 {
    t.Errorf("Expected 3, got %v", output)
  }
}

func TestErrorGettingSession(t *testing.T) {
  cookie := "session-cookie=123456; Path=/"
  request := &http.Request{Header: http.Header{"Cookie": []string{cookie}}}
  s := withSData(NewRAMStore(900), errorData{})
  _, err := s.Get(request, "session-cookie")
  if err != errSessionData {
    t.Errorf("Expected errSessionData, got %v", err)
  }
}

func TestErrorSavingSession(t *testing.T) {
  // Get new session
  request := &http.Request{}
  s := withSData(NewRAMStore(900), errorData{})
  session, err := s.Get(request, "session-cookie")
  if err != nil {
    t.Errorf("Expected no error getting session, got %v", err)
  }
  // Populate session with data and save session
  session.Values["count"] = 3
  w := &responseWriter{http.Header{}}
  err = session.Save(request, w)
  if err != errSessionData {
    t.Errorf("Expected errSessionData, got %v", err)
  }
}

type responseWriter struct {
  header http.Header
}

func (r *responseWriter) Header() http.Header {
  return r.header
}

func (r *responseWriter) Write([]byte) (int, error) {
  return 0, nil
}

func (r *responseWriter) WriteHeader(int) {
}

type errorData struct {
}

func (e errorData) GetData(
    id string) (map[interface{}]interface{}, error) {
  return nil, errSessionData
}
  
func (e errorData) SaveData(
    id string, values map[interface{}]interface{}) error {
  return errSessionData
}

func withSData(s *RAMStore, data SessionData) *RAMStore {
  result := *s
  result.SData = data
  return &result
}
