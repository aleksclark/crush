// Package e2e provides end-to-end tests for Crush TUI using the trifle framework.
//
// Run all tests:
//
//	go test -v ./e2e-next/...
//
// Update snapshots:
//
//	go test ./e2e-next/... -update
//
// Run specific test file:
//
//	go test -v ./e2e-next/... -run TestStartup
//
// Build crush binary before running tests:
//
//	go build -o crush . && go test -v ./e2e-next/...
package e2e
