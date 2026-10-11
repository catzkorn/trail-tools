package user

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/catzkorn/trail-tools/internal/authn"
	usersv1 "github.com/catzkorn/trail-tools/internal/gen/users/v1"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/users"
)

func (s *Service) GetCurrentUser(ctx context.Context, req *connect.Request[usersv1.GetCurrentUserRequest]) (*connect.Response[usersv1.GetCurrentUserResponse], error) {
	user, ok := authn.GetUser(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	switch u := user.(type) {
	case *users.WebAuthnUser:
		return connect.NewResponse(usersv1.GetCurrentUserResponse_builder{
			User: usersv1.User_builder{
				Id:   store.UUIDToString(user.ID()),
				Name: u.WebAuthnName(),
			}.Build(),
		}.Build()), nil
	default:
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unknown user type: %T", u))
	}
}
