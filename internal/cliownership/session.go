// Package cliownership marks a compiler-owned standalone process conversion.
// Its constructor is unavailable to external modules under Go's internal rule.
package cliownership

import "sync/atomic"

// Session permits immutable compiler-owned source syntax and registered
// declaration-candidate topology during standalone emission. The GlobalScope
// graph, package/file registration, ordered named members, ClassScope and Class
// definition identities, parsed declaration/root identities, OriginalName and
// Enclosing links are frozen after resolution. It does not authorize caching
// lookup decisions, resolved owners, binders, imports or generated names.
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
