// Package auth implements the AuthService: passkey registration and sign in,
// and invitations.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/catzkorn/trail-tools/internal/authn"
	authv1 "github.com/catzkorn/trail-tools/internal/gen/auth/v1"
	"github.com/catzkorn/trail-tools/internal/gen/auth/v1/authv1connect"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/users"
	"github.com/go-passkeys/go-passkeys/webauthn"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"gitlab.com/greyxor/slogor"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ authv1connect.AuthServiceHandler = (*Service)(nil)

const (
	// challengeTTL is how long a browser has to complete a ceremony.
	challengeTTL = 5 * time.Minute
	// invitationTTL is how long invitations are valid.
	invitationTTL = 7 * 24 * time.Hour
	// sessionTTL is how long users stay signed in.
	sessionTTL = 30 * 24 * time.Hour
	// maxNameLength limits user names and invitation notes.
	maxNameLength = 100
	// invitePath is the path of the frontend page for invitations.
	invitePath = "/invite/"
)

// Repository stores users, passkeys and invitations.
type Repository interface {
	HasAdmin(ctx context.Context) (bool, error)
	GetUser(ctx context.Context, userID pgtype.UUID) (users.User, error)
	GetPasskeyUser(ctx context.Context, userID pgtype.UUID) (*users.PasskeyUser, error)
	CreateChallenge(ctx context.Context, c *users.Challenge, expiry time.Time) (pgtype.UUID, error)
	TakeChallenge(ctx context.Context, id pgtype.UUID, ceremony users.Ceremony) (*users.Challenge, error)
	CreateInvitation(ctx context.Context, createdBy pgtype.UUID, kind users.InvitationKind, userID pgtype.UUID, note string, expiry time.Time) (*users.Invitation, string, error)
	GetInvitationByToken(ctx context.Context, token string) (*users.Invitation, error)
	ListInvitations(ctx context.Context, createdBy pgtype.UUID) ([]*users.Invitation, error)
	DeleteInvitation(ctx context.Context, createdBy, id pgtype.UUID) error
	ListPasskeys(ctx context.Context, userID pgtype.UUID) ([]*users.Passkey, error)
	ListPasskeyIDs(ctx context.Context, userID pgtype.UUID) ([][]byte, error)
	GetPasskey(ctx context.Context, credentialID []byte) (*users.Passkey, error)
	UsePasskey(ctx context.Context, credentialID []byte, signCount uint32, backedUp bool) error
	DeletePasskey(ctx context.Context, userID pgtype.UUID, credentialID []byte) error
	RegisterPasskey(ctx context.Context, reg *users.Registration) (users.User, error)
	CreateSession(ctx context.Context, userID pgtype.UUID, expiry time.Time) (string, error)
}

// Service implements API handlers for the auth service.
type Service struct {
	log            *slog.Logger
	repo           Repository
	rp             *webauthn.RelyingParty
	rpName         string
	baseURL        string
	bootstrapToken string
	nowFn          func() time.Time
}

// NewService creates a new Service. baseURL is the public URL of the
// application, which determines the WebAuthn relying party. bootstrapToken,
// if set, works as an invitation to create the first administrator while
// there are no administrators.
func NewService(log *slog.Logger, repo Repository, baseURL, bootstrapToken string) (*Service, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid base URL %q", baseURL)
	}
	return &Service{
		log:  log,
		repo: repo,
		rp: &webauthn.RelyingParty{
			ID:     u.Hostname(),
			Origin: u.Scheme + "://" + u.Host,
		},
		rpName:         "Trail Tools",
		baseURL:        strings.TrimSuffix(baseURL, "/"),
		bootstrapToken: bootstrapToken,
		nowFn:          time.Now,
	}, nil
}

func (s *Service) GetInvitation(ctx context.Context, req *connect.Request[authv1.GetInvitationRequest]) (*connect.Response[authv1.GetInvitationResponse], error) {
	token := req.Msg.GetToken()
	bootstrap, err := s.isBootstrap(ctx, token)
	if err != nil {
		return nil, s.internalError(ctx, "failed to check bootstrap token", err)
	}
	if bootstrap {
		return connect.NewResponse(authv1.GetInvitationResponse_builder{
			Invitation: authv1.Invitation_builder{
				Kind:      authv1.InvitationKind_INVITATION_KIND_NEW_USER,
				Note:      "Create the administrator account",
				Bootstrap: true,
			}.Build(),
		}.Build()), nil
	}
	inv, err := s.repo.GetInvitationByToken(ctx, token)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to get invitation")
	}
	return connect.NewResponse(authv1.GetInvitationResponse_builder{Invitation: invitationToProto(inv)}.Build()), nil
}

func (s *Service) CreateInvitation(ctx context.Context, req *connect.Request[authv1.CreateInvitationRequest]) (*connect.Response[authv1.CreateInvitationResponse], error) {
	user, ok := authn.GetUser(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	note := strings.TrimSpace(req.Msg.GetNote())
	if len(note) > maxNameLength {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("note must be at most %d characters", maxNameLength))
	}
	var kind users.InvitationKind
	var target pgtype.UUID
	switch req.Msg.GetKind() {
	case authv1.InvitationKind_INVITATION_KIND_NEW_USER:
		if !user.IsAdmin() {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("only administrators can invite new users"))
		}
		kind = users.InvitationNewUser
	case authv1.InvitationKind_INVITATION_KIND_ADD_PASSKEY:
		kind, target = users.InvitationAddPasskey, user.ID()
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("kind is required"))
	}
	inv, token, err := s.repo.CreateInvitation(ctx, user.ID(), kind, target, note, s.nowFn().Add(invitationTTL))
	if err != nil {
		return nil, s.internalError(ctx, "failed to create invitation", err)
	}
	return connect.NewResponse(authv1.CreateInvitationResponse_builder{
		Invitation: invitationToProto(inv),
		Url:        s.baseURL + invitePath + token,
	}.Build()), nil
}

func (s *Service) ListInvitations(ctx context.Context, req *connect.Request[authv1.ListInvitationsRequest]) (*connect.Response[authv1.ListInvitationsResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	invitations, err := s.repo.ListInvitations(ctx, userID)
	if err != nil {
		return nil, s.internalError(ctx, "failed to list invitations", err)
	}
	protoInvitations := make([]*authv1.Invitation, len(invitations))
	for i, inv := range invitations {
		protoInvitations[i] = invitationToProto(inv)
	}
	return connect.NewResponse(authv1.ListInvitationsResponse_builder{Invitations: protoInvitations}.Build()), nil
}

func (s *Service) DeleteInvitation(ctx context.Context, req *connect.Request[authv1.DeleteInvitationRequest]) (*connect.Response[authv1.DeleteInvitationResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	id, err := store.StringToUUID(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid invitation ID: %w", err))
	}
	if err := s.repo.DeleteInvitation(ctx, userID, id); err != nil {
		return nil, s.toConnectError(ctx, err, "failed to delete invitation")
	}
	return connect.NewResponse(authv1.DeleteInvitationResponse_builder{}.Build()), nil
}

func (s *Service) ListPasskeys(ctx context.Context, req *connect.Request[authv1.ListPasskeysRequest]) (*connect.Response[authv1.ListPasskeysResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	passkeys, err := s.repo.ListPasskeys(ctx, userID)
	if err != nil {
		return nil, s.internalError(ctx, "failed to list passkeys", err)
	}
	protoPasskeys := make([]*authv1.Passkey, len(passkeys))
	for i, p := range passkeys {
		b := authv1.Passkey_builder{
			Id:         base64.RawURLEncoding.EncodeToString(p.ID),
			Name:       p.Name,
			BackedUp:   p.BackedUp,
			CreateTime: timestamppb.New(p.CreateTime),
		}
		if !p.LastUsedTime.IsZero() {
			b.LastUsedTime = timestamppb.New(p.LastUsedTime)
		}
		protoPasskeys[i] = b.Build()
	}
	return connect.NewResponse(authv1.ListPasskeysResponse_builder{Passkeys: protoPasskeys}.Build()), nil
}

func (s *Service) DeletePasskey(ctx context.Context, req *connect.Request[authv1.DeletePasskeyRequest]) (*connect.Response[authv1.DeletePasskeyResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	id, err := base64.RawURLEncoding.DecodeString(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid passkey ID"))
	}
	if err := s.repo.DeletePasskey(ctx, userID, id); err != nil {
		return nil, s.toConnectError(ctx, err, "failed to delete passkey")
	}
	return connect.NewResponse(authv1.DeletePasskeyResponse_builder{}.Build()), nil
}

func (s *Service) BeginPasskeyRegistration(ctx context.Context, req *connect.Request[authv1.BeginPasskeyRegistrationRequest]) (*connect.Response[authv1.BeginPasskeyRegistrationResponse], error) {
	challenge := &users.Challenge{Ceremony: users.CeremonyRegistration, Challenge: randomBytes(32)}
	var target pgtype.UUID
	token := req.Msg.GetInvitationToken()
	bootstrap, err := s.isBootstrap(ctx, token)
	if err != nil {
		return nil, s.internalError(ctx, "failed to check bootstrap token", err)
	}
	switch {
	case bootstrap:
		challenge.Bootstrap = true
	case token != "":
		inv, err := s.repo.GetInvitationByToken(ctx, token)
		if err != nil {
			return nil, s.toConnectError(ctx, err, "failed to get invitation")
		}
		challenge.InvitationID = inv.ID
		if inv.Kind == users.InvitationAddPasskey {
			target = inv.UserID
		}
	default:
		userID, ok := authn.GetUserID(ctx)
		if !ok {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("an invitation is required to register"))
		}
		challenge.UserID = userID
		target = userID
	}

	resp := authv1.BeginPasskeyRegistrationResponse_builder{
		Challenge: challenge.Challenge,
		RpId:      s.rp.ID,
		RpName:    s.rpName,
	}
	if target.Valid {
		// Use the existing user's handle, so authenticators know the new
		// passkey is for the same account.
		challenge.UserHandle, resp.UserName, err = s.userHandle(ctx, target)
		if err != nil {
			return nil, err
		}
		if resp.ExcludeCredentialIds, err = s.repo.ListPasskeyIDs(ctx, target); err != nil {
			return nil, s.internalError(ctx, "failed to list passkeys", err)
		}
	} else {
		challenge.UserHandle = randomBytes(32)
	}
	resp.UserHandle = challenge.UserHandle
	id, err := s.repo.CreateChallenge(ctx, challenge, s.nowFn().Add(challengeTTL))
	if err != nil {
		return nil, s.internalError(ctx, "failed to create challenge", err)
	}
	resp.ChallengeId = store.UUIDToString(id)
	return connect.NewResponse(resp.Build()), nil
}

func (s *Service) FinishPasskeyRegistration(ctx context.Context, req *connect.Request[authv1.FinishPasskeyRegistrationRequest]) (*connect.Response[authv1.FinishPasskeyRegistrationResponse], error) {
	challenge, err := s.takeChallenge(ctx, req.Msg.GetChallengeId(), users.CeremonyRegistration)
	if err != nil {
		return nil, err
	}
	att, err := s.rp.VerifyAttestation(challenge.Challenge, req.Msg.GetClientDataJson(), req.Msg.GetAttestationObject())
	if err != nil {
		s.log.DebugContext(ctx, "Invalid passkey registration", slogor.Err(err))
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("the passkey could not be verified"))
	}
	if !att.Flags.UserPresent() || !att.Flags.UserVerified() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("the passkey must verify the user, e.g. with a fingerprint, face or PIN"))
	}
	publicKey, err := x509.MarshalPKIXPublicKey(att.PublicKey)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported public key: %w", err))
	}
	name, ok := att.AAGUID.Name()
	if !ok {
		name = "Passkey"
	}
	userName := strings.TrimSpace(req.Msg.GetUserName())
	if len(userName) > maxNameLength {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name must be at most %d characters", maxNameLength))
	}
	if userName == "" {
		userName = "Trail runner"
	}
	user, err := s.repo.RegisterPasskey(ctx, &users.Registration{
		Challenge: challenge,
		UserName:  userName,
		Passkey: &users.Passkey{
			ID:             att.CredentialID,
			Name:           name,
			PublicKey:      publicKey,
			Algorithm:      int32(att.Algorithm),
			AAGUID:         att.AAGUID[:],
			Transports:     req.Msg.GetTransports(),
			BackupEligible: att.Flags.BackupEligible(),
			BackedUp:       att.Flags.BackedUp(),
			SignCount:      att.Counter,
		},
	})
	if err != nil {
		if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("this passkey is already registered"))
		}
		return nil, s.toConnectError(ctx, err, "failed to register passkey")
	}
	resp := connect.NewResponse(authv1.FinishPasskeyRegistrationResponse_builder{}.Build())
	if err := s.signIn(ctx, resp.Header(), user.ID()); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) BeginPasskeyLogin(ctx context.Context, req *connect.Request[authv1.BeginPasskeyLoginRequest]) (*connect.Response[authv1.BeginPasskeyLoginResponse], error) {
	challenge := &users.Challenge{Ceremony: users.CeremonyLogin, Challenge: randomBytes(32)}
	id, err := s.repo.CreateChallenge(ctx, challenge, s.nowFn().Add(challengeTTL))
	if err != nil {
		return nil, s.internalError(ctx, "failed to create challenge", err)
	}
	return connect.NewResponse(authv1.BeginPasskeyLoginResponse_builder{
		ChallengeId: store.UUIDToString(id),
		Challenge:   challenge.Challenge,
		RpId:        s.rp.ID,
	}.Build()), nil
}

func (s *Service) FinishPasskeyLogin(ctx context.Context, req *connect.Request[authv1.FinishPasskeyLoginRequest]) (*connect.Response[authv1.FinishPasskeyLoginResponse], error) {
	challenge, err := s.takeChallenge(ctx, req.Msg.GetChallengeId(), users.CeremonyLogin)
	if err != nil {
		return nil, err
	}
	failed := connect.NewError(connect.CodeUnauthenticated, errors.New("the passkey could not be verified"))
	passkey, err := s.repo.GetPasskey(ctx, req.Msg.GetCredentialId())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("this passkey isn't registered"))
		}
		return nil, s.internalError(ctx, "failed to get passkey", err)
	}
	if handle := req.Msg.GetUserHandle(); len(handle) > 0 {
		expected, _, err := s.userHandle(ctx, passkey.UserID)
		if err != nil {
			return nil, err
		}
		if subtle.ConstantTimeCompare(handle, expected) != 1 {
			return nil, failed
		}
	}
	publicKey, err := x509.ParsePKIXPublicKey(passkey.PublicKey)
	if err != nil {
		return nil, s.internalError(ctx, "failed to parse stored public key", err)
	}
	assertion, err := s.rp.VerifyAssertion(publicKey, webauthn.Algorithm(passkey.Algorithm), challenge.Challenge,
		req.Msg.GetClientDataJson(), req.Msg.GetAuthenticatorData(), req.Msg.GetSignature())
	if err != nil {
		s.log.DebugContext(ctx, "Invalid passkey assertion", slogor.Err(err))
		return nil, failed
	}
	if !assertion.Flags.UserPresent() || !assertion.Flags.UserVerified() {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("the passkey must verify the user, e.g. with a fingerprint, face or PIN"))
	}
	// Authenticators that count signatures must always increase the count;
	// otherwise the passkey may have been cloned.
	if (assertion.Counter != 0 || passkey.SignCount != 0) && assertion.Counter <= passkey.SignCount {
		s.log.WarnContext(ctx, "Passkey signature counter did not increase", slog.String("user_id", store.UUIDToString(passkey.UserID)))
		return nil, failed
	}
	if err := s.repo.UsePasskey(ctx, passkey.ID, assertion.Counter, assertion.Flags.BackedUp()); err != nil {
		return nil, s.internalError(ctx, "failed to update passkey", err)
	}
	resp := connect.NewResponse(authv1.FinishPasskeyLoginResponse_builder{}.Build())
	if err := s.signIn(ctx, resp.Header(), passkey.UserID); err != nil {
		return nil, err
	}
	return resp, nil
}

// isBootstrap returns whether the token is the admin bootstrap token and
// there are no administrators yet.
func (s *Service) isBootstrap(ctx context.Context, token string) (bool, error) {
	if s.bootstrapToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(s.bootstrapToken)) != 1 {
		return false, nil
	}
	hasAdmin, err := s.repo.HasAdmin(ctx)
	if err != nil {
		return false, err
	}
	return !hasAdmin, nil
}

// userHandle returns the WebAuthn user handle and name of an existing user.
// Users who didn't sign up with a passkey use their ID as their handle.
func (s *Service) userHandle(ctx context.Context, userID pgtype.UUID) ([]byte, string, error) {
	user, err := s.repo.GetPasskeyUser(ctx, userID)
	switch {
	case err == nil:
		return user.UserHandle, user.Name, nil
	case errors.Is(err, store.ErrNotFound):
		return userID.Bytes[:], "Trail Tools user", nil
	default:
		return nil, "", s.internalError(ctx, "failed to get user", err)
	}
}

func (s *Service) takeChallenge(ctx context.Context, id string, ceremony users.Ceremony) (*users.Challenge, error) {
	challengeID, err := store.StringToUUID(id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid challenge ID"))
	}
	challenge, err := s.repo.TakeChallenge(ctx, challengeID, ceremony)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("the request expired, please try again"))
		}
		return nil, s.internalError(ctx, "failed to get challenge", err)
	}
	return challenge, nil
}

// signIn creates a session for the user and sets the session cookie.
func (s *Service) signIn(ctx context.Context, header http.Header, userID pgtype.UUID) error {
	expiry := s.nowFn().Add(sessionTTL)
	sessionID, err := s.repo.CreateSession(ctx, userID, expiry)
	if err != nil {
		return s.internalError(ctx, "failed to create session", err)
	}
	header.Add("Set-Cookie", authn.SessionCookie(sessionID, expiry).String())
	return nil
}

func invitationToProto(inv *users.Invitation) *authv1.Invitation {
	b := authv1.Invitation_builder{
		Id:            store.UUIDToString(inv.ID),
		Kind:          authv1.InvitationKind_INVITATION_KIND_NEW_USER,
		Note:          inv.Note,
		CreatedByName: inv.CreatedByName,
		UserName:      inv.UserName,
		CreateTime:    timestamppb.New(inv.CreateTime),
		ExpireTime:    timestamppb.New(inv.ExpireTime),
	}
	if inv.Kind == users.InvitationAddPasskey {
		b.Kind = authv1.InvitationKind_INVITATION_KIND_ADD_PASSKEY
	}
	if !inv.UseTime.IsZero() {
		b.UseTime = timestamppb.New(inv.UseTime)
	}
	return b.Build()
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	// crypto/rand.Read never returns an error.
	_, _ = rand.Read(b)
	return b
}

func (s *Service) toConnectError(ctx context.Context, err error, msg string) error {
	switch {
	case errors.Is(err, users.ErrInvitationInvalid):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, users.ErrLastPasskey):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, store.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, errors.New("not found"))
	default:
		return s.internalError(ctx, msg, err)
	}
}

func (s *Service) internalError(ctx context.Context, msg string, err error) error {
	s.log.ErrorContext(ctx, msg, slogor.Err(err))
	return connect.NewError(connect.CodeInternal, fmt.Errorf("%s: %w", msg, err))
}
