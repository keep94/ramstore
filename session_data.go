// Copyright 2013 Travis Keep. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or
// at http://opensource.org/licenses/BSD-3-Clause.

package ramstore

import (
	"sync"
	"time"
)

// RAMSessions stores session data. Session data for a particular session
// expires after a set time of inactivity for that session. RAMSessions can
// be safely used with multiple goroutines. Clients should not use this type
// directly, but should use RAMStore instead.
type RAMSessions struct {
	// In addition to the fields of this struct, mutex protects the contents of
	// the data map as well as the fields of each ramSession struct, but it
	// does not protect the contents of the map in each ramSession struct.
	// Therefore, goroutines must treat the contents of these maps as frozen.
	mutex  sync.Mutex
	data   map[string]*ramSession
	clock  func() int64
	maxAge int64
}

// NewRAMSessions creates a new RAMSessions instance. maxAge is the maximum
// allowed inactivity in seconds before data for a particular session expires.
func NewRAMSessions(maxAge int) *RAMSessions {
	return newRAMSessionsForTesting(maxAge, nowInSeconds)
}

func newRAMSessionsForTesting(maxAge int, clock func() int64) *RAMSessions {
	result := &RAMSessions{
		data:   make(map[string]*ramSession),
		clock:  clock,
		maxAge: int64(maxAge)}
	go func() {
		for {
			<-time.After(time.Duration(maxAge) * time.Second)
			result.Purge()
		}
	}()
	return result
}

// Get returns a shallow copy of the session data for a particular session ID.
// Get returns nil if the session ID does not exist or if the session data
// for the session ID expired from too much inactivity.
func (r *RAMSessions) Get(id string) map[interface{}]interface{} {
	result := r.get(id, true)
	if result == nil {
		return nil
	}
	return copyMap(result)
}

// GetData returns Get(id), nil
func (r *RAMSessions) GetData(id string) (map[interface{}]interface{}, error) {
	return r.Get(id), nil
}

// Poll works just like Get except that calling Poll does not keep session
// data from expiring.
func (r *RAMSessions) Poll(id string) map[interface{}]interface{} {
	result := r.get(id, false)
	if result == nil {
		return nil
	}
	return copyMap(result)
}

// Save saves new session data for a particular session ID.
// Save makes a shallow copy of data before saving it.
func (r *RAMSessions) Save(id string, data map[interface{}]interface{}) {
	data = copyMap(data)
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.data[id] = &ramSession{data, r.clock()}
}

// SaveData calls Save(id, data) and returns nil.
func (r *RAMSessions) SaveData(
	id string, data map[interface{}]interface{}) error {
	r.Save(id, data)
	return nil
}

// Purge removes session data that has already expired. Clients need not call
// this manually as a separate go routine calls this periodically.
func (r *RAMSessions) Purge() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	now := r.clock()
	for k, v := range r.data {
		if v.Expired(now, r.maxAge) {
			delete(r.data, k)
		}
	}
}

// AsPoller returns a view of this instance that does not keep session data
// from expiring when it is fetched.
func (r *RAMSessions) AsPoller() SessionData {
	return poller{r}
}

func (r *RAMSessions) get(
	id string, updateLastAccessed bool) map[interface{}]interface{} {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	ramSession := r.data[id]
	if ramSession == nil {
		return nil
	}
	return ramSession.Get(r.clock(), r.maxAge, updateLastAccessed)
}

func (r *RAMSessions) lenForTesting() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return len(r.data)
}

type ramSession struct {
	data         map[interface{}]interface{}
	lastAccessed int64
}

func (r *ramSession) Get(
	now int64,
	maxAge int64,
	updateLastAccessed bool) map[interface{}]interface{} {
	if r.Expired(now, maxAge) {
		return nil
	}
	if updateLastAccessed {
		r.lastAccessed = now
	}
	return r.data
}

func (r *ramSession) Expired(now int64, maxAge int64) bool {
	return now-r.lastAccessed > maxAge
}

type poller struct {
	*RAMSessions
}

func (p poller) GetData(id string) (map[interface{}]interface{}, error) {
	return p.Poll(id), nil
}

func nowInSeconds() int64 {
	return time.Now().Unix()
}

func copyMap(data map[interface{}]interface{}) map[interface{}]interface{} {
	result := make(map[interface{}]interface{}, len(data))
	for k, v := range data {
		result[k] = v
	}
	return result
}
