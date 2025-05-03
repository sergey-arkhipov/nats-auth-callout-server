package authresponse_test

import (
	"sergey-arkhipov/nats-auth-callout-server/auth-server/auth"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/authresponse"
	"strings"
	"testing"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go/micro"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockUserRepository implements UserRepository for testing
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Get(username string) (*auth.User, bool) {
	args := m.Called(username)
	return args.Get(0).(*auth.User), args.Bool(1)
}

// MockRequest implements micro.Request for testing
type MockRequest struct {
	mock.Mock
	data    []byte
	headers map[string][]string
	subject string
}

func (m *MockRequest) Data() []byte {
	return m.data
}

func (m *MockRequest) Headers() micro.Headers {
	return micro.Headers(m.headers)
}

func (m *MockRequest) Respond(data []byte, opts ...micro.RespondOpt) error {
	args := m.Called(data, opts)
	return args.Error(0)
}

func (m *MockRequest) RespondJSON(v any, opts ...micro.RespondOpt) error {
	args := m.Called(v, opts)
	return args.Error(0)
}

func (m *MockRequest) Error(err string, description string, data []byte, opts ...micro.RespondOpt) error {
	return m.Respond(data, opts...)
}

func (m *MockRequest) Reply() string {
	return m.subject
}

func (m *MockRequest) Subject() string {
	return m.subject
}

func createTestKeyPair(t *testing.T, prefix nkeys.PrefixByte) nkeys.KeyPair {
	kp, err := nkeys.CreatePair(prefix)
	require.NoError(t, err)
	return kp
}

func TestNewHandler(t *testing.T) {
	kp := &auth.KeyPairs{}
	repo := new(MockUserRepository)

	handler := authresponse.NewHandler(kp, repo)
	assert.NotNil(t, handler)
}

func TestHandler_HandleRequest(t *testing.T) {
	// Create proper key pairs with correct prefixes
	issuerKP := createTestKeyPair(t, nkeys.PrefixByteAccount)
	serverKP := createTestKeyPair(t, nkeys.PrefixByteServer)
	userKP := createTestKeyPair(t, nkeys.PrefixByteUser)

	serverPubKey, err := serverKP.PublicKey()
	require.NoError(t, err)
	userPubKey, err := userKP.PublicKey()
	require.NoError(t, err)
	issuerPubKey, err := issuerKP.PublicKey()
	require.NoError(t, err)

	t.Logf("Server pubkey: %s (should start with N)", serverPubKey)
	t.Logf("User pubkey: %s (should start with U)", userPubKey)
	t.Logf("Issuer pubkey: %s (should start with A)", issuerPubKey)

	// Validate public key prefixes
	if !strings.HasPrefix(serverPubKey, "N") {
		t.Fatalf("Server public key has incorrect prefix: %s (expected 'N')", serverPubKey)
	}
	if !strings.HasPrefix(userPubKey, "U") {
		t.Fatalf("User public key has incorrect prefix: %s (expected 'U')", userPubKey)
	}
	if !strings.HasPrefix(issuerPubKey, "A") {
		t.Fatalf("Issuer public key has incorrect prefix: %s (expected 'A')", issuerPubKey)
	}

	keyPairs := &auth.KeyPairs{
		Issuer:  issuerKP, // Account key (prefix "A")
		HasXKey: false,
	}

	t.Run("successful authentication", func(t *testing.T) {
		repo := new(MockUserRepository)
		handler := authresponse.NewHandler(keyPairs, repo)

		testUser := &auth.User{
			Account: issuerPubKey, // Use account public key as Account
			Pass:    "password",
			Permissions: jwt.Permissions{
				Pub: jwt.Permission{Allow: []string{"test.>"}},
			},
		}
		repo.On("Get", "testuser").Run(func(args mock.Arguments) {
			t.Logf("MockUserRepository.Get called with username: %s", args.String(0))
		}).Return(testUser, true)

		// Create AuthorizationRequestClaims
		arc := jwt.NewAuthorizationRequestClaims(userPubKey)
		arc.ConnectOptions.Username = "testuser"
		arc.ConnectOptions.Password = "password"
		arc.Server = jwt.ServerID{
			ID:   issuerPubKey,
			Name: "test-server", // Empty ID to avoid validation error
		}
		arc.UserNkey = userPubKey

		t.Logf("ServerID: %+v", arc.Server)

		// Encode JWT with issuer key (account account)
		token, err := arc.Encode(serverKP)
		require.NoError(t, err, "JWT encoding failed: %v", err)

		// Mock micro request
		req := &MockRequest{
			data: []byte(token),
			headers: map[string][]string{
				"Nats-Server-Id": {"test-server"},
			},
			subject: "test.subject",
		}
		req.On("Respond", mock.Anything, mock.Anything).Return(nil)

		// Call handler
		handler.HandleRequest(req)

		// Verify expectations
		repo.AssertExpectations(t)
		req.AssertCalled(t, "Respond", mock.Anything, mock.Anything)
	})
}

func TestHandler_UserClaims(t *testing.T) {
	issuerKP := createTestKeyPair(t, nkeys.PrefixByteAccount)
	userKP := createTestKeyPair(t, nkeys.PrefixByteUser)

	issuerPubKey, err := issuerKP.PublicKey()
	require.NoError(t, err)
	userPubKey, err := userKP.PublicKey()
	require.NoError(t, err)

	t.Logf("Issuer pubkey: %s (should start with A)", issuerPubKey)
	t.Logf("User pubkey: %s (should start with U)", userPubKey)

	// keyPairs := &auth.KeyPairs{
	// 	Issuer:  issuerKP,
	// 	HasXKey: false,
	// }

	t.Run("successful user claims", func(t *testing.T) {
		repo := new(MockUserRepository)
		// handler := authresponse.NewHandler(keyPairs, repo)

		testUser := &auth.User{
			Account: issuerPubKey, // Account key for Audience/Issuer
			Pass:    "dev",
			Permissions: jwt.Permissions{
				Pub: jwt.Permission{Allow: []string{"test.>"}},
			},
		}
		repo.On("Get", "dev").Return(testUser, true)

		// Create UserClaims with user key as subject
		uc := jwt.NewUserClaims(userPubKey) // Subject = user key
		uc.Permissions = testUser.Permissions
		uc.Issuer = issuerPubKey   // Issuer = account key
		uc.Audience = issuerPubKey // Audience = account key

		// Encode with issuerKP (account key)
		token, err := uc.Encode(issuerKP)
		require.NoError(t, err, "Encoding user claims failed: %v", err)

		// Decode and verify
		decoded, err := jwt.DecodeUserClaims(token)
		require.NoError(t, err, "Decoding user claims failed: %v", err)
		require.Equal(t, userPubKey, decoded.Subject, "Expected subject to be user public key")
		require.Equal(t, issuerPubKey, decoded.Issuer, "Expected issuer to be account public key")
		require.Equal(t, issuerPubKey, decoded.Audience, "Expected audience to be account public key")
		require.Equal(t, testUser.Permissions.Pub.Allow, decoded.Permissions.Pub.Allow, "Expected permissions to match")
	})
}
