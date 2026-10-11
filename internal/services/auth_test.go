package services_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	authv1 "github.com/catzkorn/trail-tools/internal/gen/auth/v1"
	"github.com/catzkorn/trail-tools/internal/gen/auth/v1/authv1connect"
	usersv1 "github.com/catzkorn/trail-tools/internal/gen/users/v1"
	"github.com/catzkorn/trail-tools/internal/gen/users/v1/usersv1connect"
)

// browser is a client that keeps the session cookie, like a browser.
type browser struct {
	cookie string
	auth   authv1connect.AuthServiceClient
	users  usersv1connect.UserServiceClient
}

func (e *env) newBrowser() *browser {
	b := &browser{}
	interceptor := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if b.cookie != "" {
				req.Header().Set("Cookie", b.cookie)
			}
			resp, err := next(ctx, req)
			if err == nil {
				for _, c := range (&http.Response{Header: resp.Header()}).Cookies() {
					if c.Name == "session" {
						b.cookie = c.Name + "=" + c.Value
					}
				}
			}
			return resp, err
		}
	})
	b.auth = authv1connect.NewAuthServiceClient(e.srv.Client(), e.srv.URL, connect.WithInterceptors(interceptor))
	b.users = usersv1connect.NewUserServiceClient(e.srv.Client(), e.srv.URL, connect.WithInterceptors(interceptor))
	return b
}

// register registers a passkey with the authenticator and returns its
// credential ID.
func (b *browser) register(t *testing.T, a *authenticator, token, name string) ([]byte, error) {
	t.Helper()
	ctx := context.Background()
	begin, err := b.auth.BeginPasskeyRegistration(ctx, connect.NewRequest(authv1.BeginPasskeyRegistrationRequest_builder{InvitationToken: token}.Build()))
	if err != nil {
		return nil, err
	}
	if begin.Msg.GetRpId() != "localhost" {
		t.Errorf("unexpected relying party %q", begin.Msg.GetRpId())
	}
	id, clientData, attestation := a.create(begin.Msg.GetChallenge(), begin.Msg.GetUserHandle())
	_, err = b.auth.FinishPasskeyRegistration(ctx, connect.NewRequest(authv1.FinishPasskeyRegistrationRequest_builder{
		ChallengeId:       begin.Msg.GetChallengeId(),
		UserName:          name,
		ClientDataJson:    clientData,
		AttestationObject: attestation,
		Transports:        []string{"internal"},
	}.Build()))
	return id, err
}

// login signs in with a passkey.
func (b *browser) login(t *testing.T, a *authenticator, credentialID []byte) error {
	t.Helper()
	ctx := context.Background()
	begin, err := b.auth.BeginPasskeyLogin(ctx, connect.NewRequest(authv1.BeginPasskeyLoginRequest_builder{}.Build()))
	if err != nil {
		return err
	}
	clientData, authData, sig := a.get(credentialID, begin.Msg.GetChallenge())
	_, err = b.auth.FinishPasskeyLogin(ctx, connect.NewRequest(authv1.FinishPasskeyLoginRequest_builder{
		ChallengeId:       begin.Msg.GetChallengeId(),
		CredentialId:      credentialID,
		ClientDataJson:    clientData,
		AuthenticatorData: authData,
		Signature:         sig,
		UserHandle:        a.keys[string(credentialID)].userHandle,
	}.Build()))
	return err
}

func (b *browser) currentUser(t *testing.T) *usersv1.User {
	t.Helper()
	resp, err := b.users.GetCurrentUser(context.Background(), connect.NewRequest(usersv1.GetCurrentUserRequest_builder{}.Build()))
	if err != nil {
		t.Fatalf("GetCurrentUser: %v", err)
	}
	return resp.Msg.GetUser()
}

func TestPasskeysAndInvitations(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()
	laptop := newAuthenticator(t, "http://localhost:8080", "localhost")

	// Registration requires an invitation.
	anon := e.newBrowser()
	_, err := anon.register(t, laptop, "", "Stranger")
	wantCode(t, err, connect.CodeUnauthenticated)
	_, err = anon.register(t, laptop, "made-up-token", "Stranger")
	wantCode(t, err, connect.CodeFailedPrecondition)

	// The bootstrap token creates the first administrator.
	inv, err := anon.auth.GetInvitation(ctx, connect.NewRequest(authv1.GetInvitationRequest_builder{Token: bootstrapToken}.Build()))
	if err != nil || !inv.Msg.GetInvitation().GetBootstrap() {
		t.Fatalf("GetInvitation for bootstrap: %v %v", inv, err)
	}
	admin := e.newBrowser()
	adminKey, err := admin.register(t, laptop, bootstrapToken, "Johan")
	if err != nil {
		t.Fatalf("bootstrap registration: %v", err)
	}
	if u := admin.currentUser(t); u.GetName() != "Johan" || !u.GetAdmin() {
		t.Errorf("unexpected admin %v", u)
	}
	// The bootstrap token only works once.
	_, err = anon.auth.GetInvitation(ctx, connect.NewRequest(authv1.GetInvitationRequest_builder{Token: bootstrapToken}.Build()))
	wantCode(t, err, connect.CodeFailedPrecondition)

	// The admin invites a new user, who registers on their phone.
	created, err := admin.auth.CreateInvitation(ctx, connect.NewRequest(authv1.CreateInvitationRequest_builder{
		Kind: authv1.InvitationKind_INVITATION_KIND_NEW_USER,
		Note: "For crew",
	}.Build()))
	if err != nil {
		t.Fatalf("CreateInvitation: %v", err)
	}
	token, ok := strings.CutPrefix(created.Msg.GetUrl(), "http://localhost:8080/invite/")
	if !ok {
		t.Fatalf("unexpected invitation URL %q", created.Msg.GetUrl())
	}
	got, err := anon.auth.GetInvitation(ctx, connect.NewRequest(authv1.GetInvitationRequest_builder{Token: token}.Build()))
	if err != nil || got.Msg.GetInvitation().GetCreatedByName() != "Johan" || got.Msg.GetInvitation().GetNote() != "For crew" {
		t.Fatalf("GetInvitation: %v %v", got, err)
	}
	phone := newAuthenticator(t, "http://localhost:8080", "localhost")
	crew := e.newBrowser()
	crewKey, err := crew.register(t, phone, token, "Crew")
	if err != nil {
		t.Fatalf("registration with invitation: %v", err)
	}
	if u := crew.currentUser(t); u.GetName() != "Crew" || u.GetAdmin() {
		t.Errorf("unexpected crew user %v", u)
	}
	// The invitation can't be used again.
	_, err = e.newBrowser().register(t, newAuthenticator(t, "http://localhost:8080", "localhost"), token, "Someone else")
	wantCode(t, err, connect.CodeFailedPrecondition)

	// Only admins can invite new users, but anyone can add a passkey on
	// another device.
	_, err = crew.auth.CreateInvitation(ctx, connect.NewRequest(authv1.CreateInvitationRequest_builder{
		Kind: authv1.InvitationKind_INVITATION_KIND_NEW_USER,
	}.Build()))
	wantCode(t, err, connect.CodePermissionDenied)
	addDevice, err := crew.auth.CreateInvitation(ctx, connect.NewRequest(authv1.CreateInvitationRequest_builder{
		Kind: authv1.InvitationKind_INVITATION_KIND_ADD_PASSKEY,
	}.Build()))
	if err != nil {
		t.Fatalf("CreateInvitation for a device: %v", err)
	}
	tablet := newAuthenticator(t, "http://localhost:8080", "localhost")
	tabletBrowser := e.newBrowser()
	addToken := strings.TrimPrefix(addDevice.Msg.GetUrl(), "http://localhost:8080/invite/")
	if _, err := tabletBrowser.register(t, tablet, addToken, ""); err != nil {
		t.Fatalf("adding a device: %v", err)
	}
	if u := tabletBrowser.currentUser(t); u.GetName() != "Crew" {
		t.Errorf("the device was added to the wrong user: %v", u)
	}
	passkeys, err := crew.auth.ListPasskeys(ctx, connect.NewRequest(authv1.ListPasskeysRequest_builder{}.Build()))
	if err != nil || len(passkeys.Msg.GetPasskeys()) != 2 {
		t.Fatalf("ListPasskeys: %v %v", passkeys, err)
	}

	// Sign in on a new browser with a registered passkey.
	fresh := e.newBrowser()
	if err := fresh.login(t, phone, crewKey); err != nil {
		t.Fatalf("login: %v", err)
	}
	if u := fresh.currentUser(t); u.GetName() != "Crew" {
		t.Errorf("signed in as the wrong user: %v", u)
	}
	if err := e.newBrowser().login(t, laptop, adminKey); err != nil {
		t.Fatalf("admin login: %v", err)
	}

	// A replayed signature counter is rejected, as the passkey may have
	// been cloned.
	laptop.keys[string(adminKey)].counter = 0
	err = e.newBrowser().login(t, laptop, adminKey)
	wantCode(t, err, connect.CodeUnauthenticated)

	// Passkeys must verify the user.
	unverified := newAuthenticator(t, "http://localhost:8080", "localhost")
	unverified.userVerified = false
	_, err = crew.register(t, unverified, "", "")
	wantCode(t, err, connect.CodeInvalidArgument)

	// The last passkey of an account can't be deleted.
	if _, err := admin.auth.DeletePasskey(ctx, connect.NewRequest(authv1.DeletePasskeyRequest_builder{
		Id: passkeysFor(t, admin)[0].GetId(),
	}.Build())); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("deleting the last passkey: got %v", err)
	}
}

func passkeysFor(t *testing.T, b *browser) []*authv1.Passkey {
	t.Helper()
	resp, err := b.auth.ListPasskeys(context.Background(), connect.NewRequest(authv1.ListPasskeysRequest_builder{}.Build()))
	if err != nil {
		t.Fatal(err)
	}
	return resp.Msg.GetPasskeys()
}
