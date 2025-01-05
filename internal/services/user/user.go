package user

import (
	"context"

	"connectrpc.com/connect"
	usersv1 "github.com/catzkorn/trail-tools/internal/gen/users/v1"
	"github.com/catzkorn/trail-tools/internal/oidc"
	"github.com/catzkorn/trail-tools/internal/store"
)

func (s *Service) GetCurrentUser(ctx context.Context, req *connect.Request[usersv1.GetCurrentUserRequest]) (*connect.Response[usersv1.GetCurrentUserResponse], error) {
	userInfo, ok := oidc.GetUserInfo(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	user, err := s.users.GetOIDCUser(ctx, userInfo.Subject)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(usersv1.GetCurrentUserResponse_builder{
		User: usersv1.User_builder{
			Id:         store.UUIDToString(user.ID),
			Email:      userInfo.Email,
			Name:       userInfo.Name,
			GivenName:  userInfo.GivenName,
			FamilyName: userInfo.FamilyName,
			AvatarUrl:  userInfo.AvatarURL,
		}.Build(),
	}.Build()), nil
}
