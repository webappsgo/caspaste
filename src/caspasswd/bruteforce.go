// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package caspasswd

import (
	"net"
	"sync"
	"time"
)

// BruteForceProtection tracks failed login attempts per IP address
type BruteForceProtection struct {
	mu       sync.RWMutex
	attempts map[string]*loginAttempts

	maxAttempts   int
	lockoutTime   time.Duration
	cleanupPeriod time.Duration
	lastCleanup   time.Time
}

type loginAttempts struct {
	count       int
	lockedUntil time.Time
}

// NewBruteForceProtection creates a new brute force protection system
func NewBruteForceProtection(maxAttempts int, lockoutTime time.Duration) *BruteForceProtection {
	bfp := &BruteForceProtection{
		attempts:      make(map[string]*loginAttempts),
		maxAttempts:   maxAttempts,
		lockoutTime:   lockoutTime,
		cleanupPeriod: 10 * time.Minute,
		lastCleanup:   time.Now(),
	}

	// Start background cleanup goroutine
	go bfp.cleanupLoop()

	return bfp
}

// CheckBlocked returns true if the IP is currently blocked
func (bfp *BruteForceProtection) CheckBlocked(ip net.IP) bool {
	if ip == nil {
		return false
	}

	bfp.mu.RLock()
	defer bfp.mu.RUnlock()

	key := ip.String()
	attempt, exists := bfp.attempts[key]
	if !exists {
		return false
	}

	// Check if still locked
	if time.Now().Before(attempt.lockedUntil) {
		return true
	}

	return false
}

// RecordFailure records a failed login attempt
func (bfp *BruteForceProtection) RecordFailure(ip net.IP) {
	if ip == nil {
		return
	}

	bfp.mu.Lock()
	defer bfp.mu.Unlock()

	key := ip.String()
	attempt, exists := bfp.attempts[key]

	if !exists {
		attempt = &loginAttempts{}
		bfp.attempts[key] = attempt
	}

	// If currently locked, don't increment (already at max)
	if time.Now().Before(attempt.lockedUntil) {
		return
	}

	// Increment failure count
	attempt.count++

	// If exceeded max attempts, lock the IP
	if attempt.count >= bfp.maxAttempts {
		attempt.lockedUntil = time.Now().Add(bfp.lockoutTime)
	}
}

// RecordSuccess records a successful login and clears failed attempts
func (bfp *BruteForceProtection) RecordSuccess(ip net.IP) {
	if ip == nil {
		return
	}

	bfp.mu.Lock()
	defer bfp.mu.Unlock()

	key := ip.String()
	delete(bfp.attempts, key)
}

// GetRemainingLockout returns the time remaining until unlock (0 if not locked)
func (bfp *BruteForceProtection) GetRemainingLockout(ip net.IP) time.Duration {
	if ip == nil {
		return 0
	}

	bfp.mu.RLock()
	defer bfp.mu.RUnlock()

	key := ip.String()
	attempt, exists := bfp.attempts[key]
	if !exists {
		return 0
	}

	if time.Now().Before(attempt.lockedUntil) {
		return time.Until(attempt.lockedUntil)
	}

	return 0
}

// cleanupLoop periodically removes old entries
func (bfp *BruteForceProtection) cleanupLoop() {
	ticker := time.NewTicker(bfp.cleanupPeriod)
	defer ticker.Stop()

	for range ticker.C {
		bfp.cleanup()
	}
}

// cleanup removes expired lockouts and old entries
func (bfp *BruteForceProtection) cleanup() {
	bfp.mu.Lock()
	defer bfp.mu.Unlock()

	now := time.Now()
	for key, attempt := range bfp.attempts {
		// Remove if lockout has expired and no recent failures
		if now.After(attempt.lockedUntil) && attempt.count < bfp.maxAttempts {
			delete(bfp.attempts, key)
		}
		// Remove if lockout expired more than cleanup period ago
		if now.After(attempt.lockedUntil.Add(bfp.cleanupPeriod)) {
			delete(bfp.attempts, key)
		}
	}

	bfp.lastCleanup = now
}
