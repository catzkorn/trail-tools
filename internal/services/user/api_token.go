package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/catzkorn/trail-tools/internal/authn"
	usersv1 "github.com/catzkorn/trail-tools/internal/gen/users/v1"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/users"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) CreateAPIToken(ctx context.Context, req *connect.Request[usersv1.CreateAPITokenRequest]) (*connect.Response[usersv1.CreateAPITokenResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	token, secret, err := s.tokens.CreateAPIToken(ctx, userID, name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create API token: %w", err))
	}
	return connect.NewResponse(usersv1.CreateAPITokenResponse_builder{
		Token:  apiTokenToProto(token),
		Secret: secret,
	}.Build()), nil
}

func (s *Service) ListAPITokens(ctx context.Context, req *connect.Request[usersv1.ListAPITokensRequest]) (*connect.Response[usersv1.ListAPITokensResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	tokens, err := s.tokens.ListAPITokens(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list API tokens: %w", err))
	}
	protoTokens := make([]*usersv1.APIToken, len(tokens))
	for i, token := range tokens {
		protoTokens[i] = apiTokenToProto(token)
	}
	return connect.NewResponse(usersv1.ListAPITokensResponse_builder{
		Tokens: protoTokens,
	}.Build()), nil
}

func (s *Service) DeleteAPIToken(ctx context.Context, req *connect.Request[usersv1.DeleteAPITokenRequest]) (*connect.Response[usersv1.DeleteAPITokenResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	tokenID, err := store.StringToUUID(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid token ID: %w", err))
	}
	if err := s.tokens.DeleteAPIToken(ctx, userID, tokenID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("token not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete API token: %w", err))
	}
	return connect.NewResponse(usersv1.DeleteAPITokenResponse_builder{}.Build()), nil
}

func apiTokenToProto(token *users.APIToken) *usersv1.APIToken {
	b := usersv1.APIToken_builder{
		Id:         store.UUIDToString(token.ID),
		Name:       token.Name,
		CreateTime: timestamppb.New(token.CreateTime.Time),
	}
	if token.LastUsedTime.Valid {
		b.LastUsedTime = timestamppb.New(token.LastUsedTime.Time)
	}
	return b.Build()
}
