package strava

import (
	"context"
	"net/http"
	"time"

	"github.com/johanbrandhorst/trail-tools/internal/errreport"
)

// SyncPath is the path of the scheduled sync endpoint.
const SyncPath = "/internal/strava/sync"

// scheduledSyncBudget is how long a scheduled sync runs for. It must be less
// than the request timeout of the server.
const scheduledSyncBudget = 10 * time.Minute

// SyncHandler returns a handler that syncs all connections that are due a
// sync, for running from a scheduler such as Cloud Scheduler. It must be
// protected by authorization.
func SyncHandler(report *errreport.Reporter, syncer *Syncer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), scheduledSyncBudget)
		defer cancel()
		if err := syncer.SyncDue(ctx); err != nil && ctx.Err() == nil {
			report.Report(ctx, "Scheduled Strava sync failed", err)
			http.Error(w, "sync failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
