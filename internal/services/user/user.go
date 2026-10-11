package user

import (
	"context"

	"connectrpc.com/connect/v2"
	"github.com/catzkorn/trail-tools/internal/authn"
	usersv1 "github.com/catzkorn/trail-tools/internal/gen/users/v1"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/users"
)

func (s *Service) GetCurrentUser(ctx context.Context, req *usersv1.GetCurrentUserRequest) (*usersv1.GetCurrentUserResponse, error) {
	user, ok := authn.GetUser(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "")
	}
	switch u := user.(type) {
	case *users.PasskeyUser:
		return usersv1.GetCurrentUserResponse_builder{
			User: usersv1.User_builder{
				Id:    store.UUIDToString(user.ID()),
				Name:  u.Name,
				Admin: u.IsAdmin(),
			}.Build(),
		}.Build(), nil
	default:
		return nil, connect.Errorf(connect.CodeInternal, "unknown user type: %T", u)
	}
}
