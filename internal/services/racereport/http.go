package racereport

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/catzkorn/trail-tools/internal/html"
	"github.com/catzkorn/trail-tools/internal/services/trip"
	"github.com/catzkorn/trail-tools/internal/store"
	"gitlab.com/greyxor/slogor"
)

// PageHandler serves the frontend for public race reports at
// /u/<handle>/races/<slug>, with the title, description and Open Graph tags
// of the report in the HTML, and passes the other public pages on to next.
// It must be mounted at /u/ behind the authn middleware, so authors can
// preview their drafts. watch adds the live reload script.
func (s *Service) PageHandler(watch bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, trip.ProfilePath), "/"), "/")
		if len(parts) != 3 || parts[1] != RacePath {
			next.ServeHTTP(w, r)
			return
		}
		profile, report, err := s.PublicReport(r.Context(), parts[0], parts[2])
		if err != nil {
			if !errors.Is(err, store.ErrNotFound) {
				s.log.ErrorContext(r.Context(), "Failed to get public race report", slogor.Err(err))
			}
			// The frontend shows its own error page.
			html.Render(w, watch, http.StatusNotFound, html.Page{Title: "Not found · trailrunn.ing", NoIndex: true})
			return
		}
		description := trip.Excerpt(report.Sections.Summary, 200)
		if description == "" {
			description = fmt.Sprintf("%s's race report of %s on %s.", profile.DisplayName, report.RaceName, report.RaceDate.Format("2 January 2006"))
		}
		html.Render(w, watch, http.StatusOK, html.Page{
			Title:         report.Title + " · " + profile.DisplayName,
			Description:   description,
			URL:           s.ReportURL(profile.Handle, report.Slug),
			Type:          "article",
			PublishedTime: report.PublishTime,
			// Authors can preview drafts, which shouldn't be indexed.
			NoIndex: !report.Published,
		})
	})
}
