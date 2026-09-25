// Package clock is a process clock for expiry decisions: time.Now plus an
// offset that starts at zero and only moves forward, so a test build can run
// expiry in seconds instead of waiting it out. Moving the clock mints nothing:
// every credential is still earned and still expires, only sooner.
package clock
