package ramstore

import (
  "sync"
  "testing"
)

type fakeClock struct {
  mutex sync.Mutex
  now int64
}

func newFakeClock() *fakeClock {
  return &fakeClock{now: 1234567890}
}

func (c *fakeClock) Wait(seconds int64) {
  c.mutex.Lock()
  defer c.mutex.Unlock()
  c.now += seconds
}

func (c *fakeClock) Now() int64 {
  c.mutex.Lock()
  defer c.mutex.Unlock()
  return c.now
}

func (c *fakeClock) NowFunc() func() int64 {
  return func() int64 { return c.Now() }
}

func TestGetAndSave(t *testing.T) {
  c := newFakeClock()
  r := newRAMSessionsForTesting(900, c.NowFunc())
  m := map[interface{}]interface{} {5: 8}
  r.Save("key", m)
  m[5] = 10
  r.Save("key2", m)
  c.Wait(899)
  if output := r.Get("key")[5].(int); output != 8 {
    t.Errorf("Expected 8, got %v", output)
  }
  c.Wait(1)
  if output := r.Get("key2")[5].(int); output != 10 {
    t.Errorf("Expected 10, got %v", output)
  }
  c.Wait(900)
  if output := r.Get("key"); output != nil {
    t.Errorf("Expected nil, got %v", output)
  }
  if output := r.Get("key2")[5].(int); output != 10 {
    t.Errorf("Expected 10, got %v", output)
  }
  c.Wait(901)
  if output := r.Get("key"); output != nil {
    t.Errorf("Expected nil, got %v", output)
  }
  if output := r.Get("key2"); output != nil {
    t.Errorf("Expected nil, got %v", output)
  }
}

func TestGetDoesDefensiveCopy(t *testing.T) {
  c := newFakeClock()
  r := newRAMSessionsForTesting(900, c.NowFunc())
  r.Save("key", map[interface{}]interface{} {5: 8})
  m := r.Get("key")
  m[5] = 12
  if output := r.Get("key")[5].(int); output != 8 {
    t.Errorf("Expected 8, got %v", output)
  }
}

func TestPurge(t *testing.T) {
  c := newFakeClock()
  r := newRAMSessionsForTesting(900, c.NowFunc())
  r.Save("key1", nil)
  r.Save("key2", nil)
  c.Wait(1)
  r.Save("key3", nil)
  r.Purge()
  if output := r.lenForTesting(); output != 3 {
    t.Errorf("Expected 3, got %v", output)
  }
  c.Wait(900)
  r.Purge()
  if output := r.lenForTesting(); output != 1 {
    t.Errorf("Expected 1, got %v", output)
  }
}
