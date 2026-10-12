package trip

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect/v2"
	tripsv1 "github.com/catzkorn/trail-tools/internal/gen/trips/v1"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/trips"
)

func (s *Service) SetTripSections(ctx context.Context, req *tripsv1.SetTripSectionsRequest) (*tripsv1.SetTripSectionsResponse, error) {
	userID, reportID, err := s.parseReportID(ctx, req.GetReportId())
	if err != nil {
		return nil, err
	}
	sections, err := sectionsFromProto(req.GetSections())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
	}
	if _, err := s.repo.SetSections(ctx, userID, reportID, sections); err != nil {
		return nil, s.toConnectError(ctx, err, "failed to set sections")
	}
	report, err := s.repo.GetReport(ctx, userID, reportID)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to get trip report")
	}
	result, err := s.reportWithStory(ctx, report)
	if err != nil {
		return nil, err
	}
	return tripsv1.SetTripSectionsResponse_builder{Report: result}.Build(), nil
}

// sectionsFromProto validates and converts the sections of a report.
func sectionsFromProto(sections []*tripsv1.TripSection) ([]*trips.Section, error) {
	switch {
	case len(sections) == 0:
		return nil, errors.New("a report needs at least one section")
	case len(sections) > maxSections:
		return nil, fmt.Errorf("a report can have at most %d sections", maxSections)
	}
	result := make([]*trips.Section, len(sections))
	length := 0
	for i, sec := range sections {
		title := strings.TrimSpace(sec.GetTitle())
		if err := validateTitle(title); err != nil {
			return nil, err
		}
		length += utf8.RuneCountInString(sec.GetBody())
		if length > maxBodyLength {
			return nil, fmt.Errorf("the text of the sections must be at most %d characters", maxBodyLength)
		}
		result[i] = &trips.Section{Title: title, Body: sec.GetBody()}
		if sec.GetId() != "" {
			id, err := store.StringToUUID(sec.GetId())
			if err != nil {
				return nil, fmt.Errorf("invalid section ID: %w", err)
			}
			result[i].ID = id
		}
		if sec.HasStartDistanceMeters() != sec.HasEndDistanceMeters() {
			return nil, errors.New("set both the start and end distance of a section, or neither")
		}
		if sec.HasStartDistanceMeters() {
			start, end := sec.GetStartDistanceMeters(), sec.GetEndDistanceMeters()
			if math.IsNaN(start) || math.IsInf(end, 0) || start < 0 || !(end > start) {
				return nil, errors.New("a section's end distance must be after its start distance, which can't be negative")
			}
			result[i].StartDistanceMeters, result[i].EndDistanceMeters, result[i].HasDistances = start, end, true
		}
		ids, err := store.UUIDsToPG(sec.GetPhotoIds())
		if err != nil {
			return nil, err
		}
		result[i].PhotoIDs = ids
	}
	return result, nil
}
