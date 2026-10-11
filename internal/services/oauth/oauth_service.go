// Package oauth implements the OAuth service, which lets users approve OAuth
// clients and manage their connected clients.
package oauth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/catzkorn/trail-tools/internal/authn"
	oauthv1 "github.com/catzkorn/trail-tools/internal/gen/oauth/v1"
	"github.com/catzkorn/trail-tools/internal/gen/oauth/v1/oauthv1connect"
	"github.com/catzkorn/trail-tools/internal/oauth"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/users"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ oauthv1connect.OAuthServiceHandler = (*Service)(nil)

// AuthorizationServer validates and approves authorization requests.
type AuthorizationServer interface {
	ParseAuthorizationRequest(query string) (*oauth.AuthorizationRequest, error)
	Approve(ctx context.Context, userID pgtype.UUID, req *oauth.AuthorizationRequest) (string, error)
}

// GrantRepository stores the grants of connected clients.
type GrantRepository interface {
	ListOAuthGrants(ctx context.Context, userID pgtype.UUID) ([]*users.OAuthGrant, error)
	DeleteOAuthGrant(ctx context.Context, userID pgtype.UUID, grantID pgtype.UUID) error
}

// Service implements the OAuth service.
type Service struct {
	log    *slog.Logger
	server AuthorizationServer
	grants GrantRepository
}

// NewService creates a new Service.
func NewService(log *slog.Logger, server AuthorizationServer, grants GrantRepository) *Service {
	return &Service{
		log:    log,
		server: server,
		grants: grants,
	}
}

// parseRequest parses an authorization request, returning an
// InvalidArgument error if it is invalid.
func (s *Service) parseRequest(query string) (*oauth.AuthorizationRequest, error) {
	req, err := s.server.ParseAuthorizationRequest(query)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid authorization request: %w", err))
	}
	return req, nil
}

func (s *Service) GetAuthorization(ctx context.Context, req *connect.Request[oauthv1.GetAuthorizationRequest]) (*connect.Response[oauthv1.GetAuthorizationResponse], error) {
	if _, ok := authn.GetUserID(ctx); !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	authzReq, err := s.parseRequest(req.Msg.GetQuery())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(oauthv1.GetAuthorizationResponse_builder{
		ClientName:  authzReq.ClientName,
		RedirectUri: authzReq.RedirectURI,
	}.Build()), nil
}

func (s *Service) ApproveAuthorization(ctx context.Context, req *connect.Request[oauthv1.ApproveAuthorizationRequest]) (*connect.Response[oauthv1.ApproveAuthorizationResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	authzReq, err := s.parseRequest(req.Msg.GetQuery())
	if err != nil {
		return nil, err
	}
	redirectURI, err := s.server.Approve(ctx, userID, authzReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to approve authorization: %w", err))
	}
	return connect.NewResponse(oauthv1.ApproveAuthorizationResponse_builder{
		RedirectUri: redirectURI,
	}.Build()), nil
}

func (s *Service) DenyAuthorization(ctx context.Context, req *connect.Request[oauthv1.DenyAuthorizationRequest]) (*connect.Response[oauthv1.DenyAuthorizationResponse], error) {
	authzReq, err := s.parseRequest(req.Msg.GetQuery())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(oauthv1.DenyAuthorizationResponse_builder{
		RedirectUri: authzReq.Deny(),
	}.Build()), nil
}

func (s *Service) ListConnections(ctx context.Context, req *connect.Request[oauthv1.ListConnectionsRequest]) (*connect.Response[oauthv1.ListConnectionsResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	grants, err := s.grants.ListOAuthGrants(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list connections: %w", err))
	}
	connections := make([]*oauthv1.Connection, len(grants))
	for i, grant := range grants {
		b := oauthv1.Connection_builder{
			Id:          store.UUIDToString(grant.ID),
			ClientName:  grant.ClientName,
			RedirectUri: grant.RedirectURI,
			CreateTime:  timestamppb.New(grant.CreateTime.Time),
		}
		if grant.LastUsedTime.Valid {
			b.LastUsedTime = timestamppb.New(grant.LastUsedTime.Time)
		}
		connections[i] = b.Build()
	}
	return connect.NewResponse(oauthv1.ListConnectionsResponse_builder{
		Connections: connections,
	}.Build()), nil
}

func (s *Service) DeleteConnection(ctx context.Context, req *connect.Request[oauthv1.DeleteConnectionRequest]) (*connect.Response[oauthv1.DeleteConnectionResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	grantID, err := store.StringToUUID(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connection ID: %w", err))
	}
	if err := s.grants.DeleteOAuthGrant(ctx, userID, grantID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("connection not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete connection: %w", err))
	}
	return connect.NewResponse(oauthv1.DeleteConnectionResponse_builder{}.Build()), nil
}
