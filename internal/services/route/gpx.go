package route

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/catzkorn/trail-tools/internal/authn"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/store"
	"gitlab.com/greyxor/slogor"
)

// unsafeFilenameChars matches characters that are replaced in download file
// names.
var unsafeFilenameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// GPXHandler serves routes as GPX files at /routes/{id}.gpx. It must be
// mounted at /routes/ behind the authn middleware. Other paths are passed to
// next, so the frontend can serve its own /routes pages.
func (s *Service) GPXHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := strings.CutSuffix(strings.TrimPrefix(r.URL.Path, "/routes/"), ".gpx")
		if !ok || strings.Contains(id, "/") || r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}
		userID, ok := authn.GetUserID(r.Context())
		if !ok {
			http.Error(w, "sign in to download routes", http.StatusUnauthorized)
			return
		}
		routeID, err := store.StringToUUID(id)
		if err != nil {
			http.Error(w, "invalid route ID", http.StatusBadRequest)
			return
		}
		route, err := s.routes.GetRoute(r.Context(), userID, routeID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			s.log.ErrorContext(r.Context(), "Failed to get route", slogor.Err(err))
			http.Error(w, "failed to get route", http.StatusInternalServerError)
			return
		}
		points, _ := route.Path()
		filename := strings.Trim(unsafeFilenameChars.ReplaceAllString(route.Name, "-"), "-")
		if filename == "" {
			filename = "route"
		}
		w.Header().Set("Content-Type", "application/gpx+xml")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.gpx"`, filename))
		if err := geo.WriteGPX(w, route.Name, points); err != nil {
			s.log.ErrorContext(r.Context(), "Failed to write GPX", slogor.Err(err))
		}
	})
}
