// Package cliownership marks a compiler-owned standalone process conversion.
// Its constructor is unavailable to external modules under Go's internal rule.
package cliownership

import "sync/atomic"

// Session permits only immutable compiler-owned source syntax during emission.
// It does not authorize caching resolved symbols, binders, imports or names.
type Session struct{ active atomic.Bool }

func New() *Session {
	session := &Session{}
	session.active.Store(true)
	return session
}

func (session *Session) Active() bool {
	return session != nil && session.active.Load()
}

func (session *Session) Close() {
	if session != nil {
		session.active.Store(false)
	}
}
