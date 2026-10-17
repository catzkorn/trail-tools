package lactate

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect/v2"
	"github.com/johanbrandhorst/trail-tools/internal/authn"
	lactatev1 "github.com/johanbrandhorst/trail-tools/internal/gen/lactate/v1"
	"github.com/johanbrandhorst/trail-tools/internal/lactate"
	"github.com/johanbrandhorst/trail-tools/internal/store"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// SharePath is the path of recording links, followed by their token.
	// Opening one sets the recorder cookie and redirects to RecordPath.
	SharePath = "/share/lactate-tests/"
	// RecordPath is the page that records a test with a recording link.
	RecordPath = "/record/lactate-test"
	// recorderCookie holds the secret of the browser that opened a
	// recording link.
	recorderCookie = "lactate_recorder"
	// shareLifetime is how long a recording link works: long enough to
	// send it ahead of a test, short enough that a forgotten one expires.
	shareLifetime = 24 * time.Hour
)

// errNoRecording is returned to a browser without a recording.
var errNoRecording = connect.NewError(connect.CodePermissionDenied,
	"this recording link has expired or was opened in another browser; ask for a new link")

func hashSecret(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

type recorderKey struct{}

// RecorderMiddleware passes the recorder cookie of a request to the
// service's handlers, which authenticate recordings with it.
func RecorderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(recorderCookie); err == nil && cookie.Value != "" {
			r = r.WithContext(context.WithValue(r.Context(), recorderKey{}, cookie.Value))
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Service) CreateLactateTestShare(ctx context.Context, _ *lactatev1.CreateLactateTestShareRequest) (*lactatev1.CreateLactateTestShareResponse, error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "")
	}
	token := rand.Text()
	share, err := s.tests.CreateShare(ctx, userID, hashSecret(token), s.nowFn().Add(shareLifetime))
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to create lactate test share")
	}
	protoShare := shareToProto(share)
	protoShare.SetUrl(s.baseURL + SharePath + token)
	return lactatev1.CreateLactateTestShareResponse_builder{Share: protoShare}.Build(), nil
}

func (s *Service) ListLactateTestShares(ctx context.Context, _ *lactatev1.ListLactateTestSharesRequest) (*lactatev1.ListLactateTestSharesResponse, error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "")
	}
	shares, err := s.tests.ListShares(ctx, userID)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to list lactate test shares")
	}
	protoShares := make([]*lactatev1.LactateTestShare, len(shares))
	for i, share := range shares {
		protoShares[i] = shareToProto(share)
	}
	return lactatev1.ListLactateTestSharesResponse_builder{Shares: protoShares}.Build(), nil
}

func (s *Service) DeleteLactateTestShare(ctx context.Context, req *lactatev1.DeleteLactateTestShareRequest) (*lactatev1.DeleteLactateTestShareResponse, error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "")
	}
	shareID, err := store.StringToUUID(req.GetId())
	if err != nil {
		return nil, connect.Errorf(connect.CodeInvalidArgument, "invalid lactate test share ID: %v", err).WithCause(err)
	}
	if err := s.tests.DeleteShare(ctx, userID, shareID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "recording link not found")
		}
		return nil, s.toConnectError(ctx, err, "failed to delete lactate test share")
	}
	return lactatev1.DeleteLactateTestShareResponse_builder{}.Build(), nil
}

func (s *Service) GetLactateTestRecording(ctx context.Context, _ *lactatev1.GetLactateTestRecordingRequest) (*lactatev1.GetLactateTestRecordingResponse, error) {
	recording, err := s.recording(ctx)
	if err != nil {
		return nil, err
	}
	resp := lactatev1.GetLactateTestRecordingResponse_builder{
		OwnerName:  recording.OwnerName,
		ExpireTime: timestamppb.New(recording.ExpireTime),
	}.Build()
	if recording.TestID.Valid {
		test, err := s.tests.GetTest(ctx, recording.UserID, recording.TestID)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return nil, s.toConnectError(ctx, err, "failed to get the recorded lactate test")
		}
		if test != nil {
			resp.SetTest(TestToProto(test))
		}
	}
	return resp, nil
}

func (s *Service) RecordLactateTest(ctx context.Context, req *lactatev1.RecordLactateTestRequest) (*lactatev1.RecordLactateTestResponse, error) {
	recording, err := s.recording(ctx)
	if err != nil {
		return nil, err
	}
	test, err := testFromProto(recording.UserID, req.GetTest())
	if err != nil {
		return nil, err
	}
	var saved *lactate.Test
	if recording.TestID.Valid {
		test.ID = recording.TestID
		saved, err = s.tests.UpdateTest(ctx, test)
	}
	// Record a new test with the first save, or if the user deleted the
	// recorded one.
	if !recording.TestID.Valid || errors.Is(err, store.ErrNotFound) {
		saved, err = s.tests.CreateTest(ctx, test)
		if err == nil {
			err = s.tests.SetShareTest(ctx, recording.ID, saved.ID)
		}
	}
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to record lactate test")
	}
	return lactatev1.RecordLactateTestResponse_builder{Test: TestToProto(saved)}.Build(), nil
}

// recording returns the recording of the request's browser.
func (s *Service) recording(ctx context.Context) (*lactate.Recording, error) {
	secret, ok := ctx.Value(recorderKey{}).(string)
	if !ok {
		return nil, errNoRecording
	}
	recording, err := s.tests.GetRecording(ctx, hashSecret(secret))
	if errors.Is(err, store.ErrNotFound) {
		return nil, errNoRecording
	}
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to get lactate test recording")
	}
	return recording, nil
}

// ShareHandler opens recording links: the first browser to open one gets a
// cookie with a new secret, which records with it until it expires, and is
// redirected to the recording page. Other browsers, and expired links, are
// redirected too, where the page says the link doesn't work.
func (s *Service) ShareHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.URL.Path, SharePath)
		share, err := s.tests.GetShareByToken(r.Context(), hashSecret(token))
		switch {
		case errors.Is(err, store.ErrNotFound):
		case err != nil:
			s.report.Report(r.Context(), "Failed to open a lactate test share", err)
			http.Error(w, "failed to open the recording link", http.StatusInternalServerError)
			return
		case share.RecorderSHA256 != nil:
			// The browser that opened the link may open it again; it still has
			// its cookie.
		default:
			secret := rand.Text()
			opened, err := s.tests.OpenShare(r.Context(), share.ID, hashSecret(secret))
			if err != nil {
				s.report.Report(r.Context(), "Failed to open a lactate test share", err)
				http.Error(w, "failed to open the recording link", http.StatusInternalServerError)
				return
			}
			if opened {
				http.SetCookie(w, &http.Cookie{
					Name:     recorderCookie,
					Value:    secret,
					Path:     "/",
					Expires:  share.ExpireTime,
					Secure:   true,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}
		}
		http.Redirect(w, r, RecordPath, http.StatusSeeOther)
	})
}

func shareToProto(share *lactate.Share) *lactatev1.LactateTestShare {
	return lactatev1.LactateTestShare_builder{
		Id:         store.UUIDToString(share.ID),
		Opened:     share.RecorderSHA256 != nil,
		TestId:     store.UUIDToString(share.TestID),
		ExpireTime: timestamppb.New(share.ExpireTime),
		CreateTime: timestamppb.New(share.CreateTime),
	}.Build()
}
