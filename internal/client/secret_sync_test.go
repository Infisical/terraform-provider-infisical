package infisicalclient

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

// A 429 with a long Retry-After must not keep the read waiting after ctx ends. The secret sync waiter relies on
// this to keep within timeouts.create and timeouts.update.
func TestGetSecretSyncByIdWithContextStopsRetryWaitAtDeadline(t *testing.T) {
	var calls atomic.Int32
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	configureRetries(client.Config.HttpClient)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.GetSecretSyncByIdWithContext(ctx, GetSecretSyncByIdRequest{App: SecretSyncAppSpacelift, ID: "sync-id"})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error when the context ends during the retry wait")
	}
	if elapsed > 5*time.Second {
		t.Errorf("expected the read to stop at the deadline, it took %s", elapsed)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("expected one request before the deadline, got %d", got)
	}
}
