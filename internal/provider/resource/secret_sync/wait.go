package resource

import (
	"context"
	"errors"
	"fmt"
	"time"

	infisical "terraform-provider-infisical/internal/client"
)

const (
	defaultSecretSyncWaitTimeout     = 10 * time.Minute
	defaultSecretSyncWaitTimeoutText = "10m" // for the docs; the Duration prints "10m0s"
	// Infisical can take about a minute to settle a sync that it retries: the job can start up to 10s after it is
	// queued, and a failing sync runs 5 attempts with 3s, 6s, 12s and 24s between them.
	minSecretSyncWaitTimeout     = 1 * time.Minute
	minSecretSyncWaitTimeoutText = "1m"
	// Status reads share the API's read rate limit, so polling backs off.
	defaultSecretSyncPollInitialInterval = 5 * time.Second
	defaultSecretSyncPollMaxInterval     = 30 * time.Second
	// The wait context has expired by the final read, so it gets its own short budget.
	secretSyncFinalReadTimeout = 15 * time.Second
)

// Infisical syncs in a background queue: syncStatus goes pending, running, then succeeded or failed.

func syncStatusString(secretSync infisical.SecretSync) string {
	if secretSync.SyncStatus == nil {
		return "none"
	}
	return string(*secretSync.SyncStatus)
}

func isSyncStatusSettled(secretSync infisical.SecretSync) bool {
	if secretSync.SyncStatus == nil {
		return true
	}
	return *secretSync.SyncStatus == infisical.SecretSyncStatusSucceeded || *secretSync.SyncStatus == infisical.SecretSyncStatusFailed
}

// syncPollIntervals returns the first interval and its cap. Tests set a fixed pollInterval.
func (r *SecretSyncBaseResource) syncPollIntervals() (time.Duration, time.Duration) {
	if r.pollInterval > 0 {
		return r.pollInterval, r.pollInterval
	}
	return defaultSecretSyncPollInitialInterval, defaultSecretSyncPollMaxInterval
}

// nextSyncPollWait returns the first interval, then doubles the previous wait up to the cap.
func nextSyncPollWait(previous, initial, maxInterval time.Duration) time.Duration {
	if previous <= 0 {
		return initial
	}
	return min(previous*2, maxInterval)
}

// syncPoll is one wait for a secret sync to reach a status.
type syncPoll struct {
	r          *SecretSyncBaseResource
	id         string
	waitingFor string
	done       func(infisical.SecretSync) bool
	lastStatus string
}

// read gets the sync. Errors are sentence fragments for the caller's diagnostic.
func (p *syncPoll) read(ctx context.Context) (infisical.SecretSync, error) {
	secretSync, err := p.r.client.GetSecretSyncByIdWithContext(ctx, infisical.GetSecretSyncByIdRequest{App: p.r.App, ID: p.id})
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			return infisical.SecretSync{}, fmt.Errorf("the secret sync was deleted while Terraform was waiting for %s", p.waitingFor)
		}
		return infisical.SecretSync{}, fmt.Errorf("couldn't read the secret sync while waiting for %s: %w", p.waitingFor, err)
	}
	return secretSync, nil
}

func (p *syncPoll) timedOut(status string) error {
	return fmt.Errorf("timed out waiting for %s (status: %s). The sync keeps running in Infisical. To wait longer, increase the values in timeouts", p.waitingFor, status)
}

// ended reports a wait whose ctx was cancelled or reached its deadline.
func (p *syncPoll) ended(ctx context.Context) (infisical.SecretSync, error) {
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return infisical.SecretSync{}, fmt.Errorf("stopped waiting for %s because the operation was cancelled (last status: %s)", p.waitingFor, p.lastStatus)
	}
	// The last read can be up to one backoff interval old: read once more, so a sync that finished by the
	// deadline counts and the error shows the current status.
	finalCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), secretSyncFinalReadTimeout)
	defer cancel()
	secretSync, err := p.read(finalCtx)
	if err != nil {
		if finalCtx.Err() != nil {
			return infisical.SecretSync{}, p.timedOut(p.lastStatus)
		}
		return infisical.SecretSync{}, err
	}
	if p.done(secretSync) {
		return secretSync, nil
	}
	return secretSync, p.timedOut(syncStatusString(secretSync))
}

// pollSecretSync reads the sync until done returns true or ctx ends. Without readFirst it waits one interval first.
// Reads use ctx, so a stalled or rate-limited read can't outlast the timeout.
func (r *SecretSyncBaseResource) pollSecretSync(ctx context.Context, id string, waitingFor string, readFirst bool, done func(infisical.SecretSync) bool) (infisical.SecretSync, error) {
	p := &syncPoll{r: r, id: id, waitingFor: waitingFor, done: done, lastStatus: "not read yet"}
	initial, maxInterval := r.syncPollIntervals()

	var wait time.Duration
	if !readFirst {
		wait = initial
	}

	for {
		if wait > 0 {
			select {
			case <-ctx.Done():
				return p.ended(ctx)
			case <-time.After(wait):
			}
		}

		secretSync, err := p.read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return p.ended(ctx)
			}
			return infisical.SecretSync{}, err
		}

		if done(secretSync) {
			return secretSync, nil
		}

		p.lastStatus = syncStatusString(secretSync)
		wait = nextSyncPollWait(wait, initial, maxInterval)
	}
}

// waitForSyncIdle waits until no sync job is pending or running, so a job that loaded the old configuration can't
// report its result as ours. "pending" can hide a running job, so only a settled status counts.
func (r *SecretSyncBaseResource) waitForSyncIdle(ctx context.Context, id string) error {
	_, err := r.pollSecretSync(ctx, id, "the sync that was already in progress to finish", true, isSyncStatusSettled)
	return err
}

// waitForSyncResult waits for the sync that a create or update queued, and returns an error if it failed.
func (r *SecretSyncBaseResource) waitForSyncResult(ctx context.Context, id string) error {
	secretSync, err := r.pollSecretSync(ctx, id, "the sync to finish", false, func(secretSync infisical.SecretSync) bool {
		return secretSync.SyncStatus != nil && isSyncStatusSettled(secretSync)
	})
	if err != nil {
		return err
	}

	if *secretSync.SyncStatus == infisical.SecretSyncStatusFailed {
		message := "no error message was returned"
		if secretSync.LastSyncMessage != nil && *secretSync.LastSyncMessage != "" {
			message = *secretSync.LastSyncMessage
		}
		return fmt.Errorf("the sync failed: %s", message)
	}

	return nil
}
