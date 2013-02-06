package ramstore

import (
  "net/http"
  "testing"
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
