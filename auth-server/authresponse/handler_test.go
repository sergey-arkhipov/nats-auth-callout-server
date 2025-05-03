package authresponse_test

import (
	"sergey-arkhipov/nats-auth-callout-server/auth-server/auth"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/authresponse"
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

	keyPairs := &auth.KeyPairs{
		Issuer:  issuerKP,
		HasXKey: false,
	}

	t.Run("successful authentication", func(t *testing.T) {
		repo := new(MockUserRepository)
		handler := authresponse.NewHandler(keyPairs, repo)

		testUser := &auth.User{
			Account: "TEST",
			Pass:    "password",
			Permissions: jwt.Permissions{
				Pub: jwt.Permission{Allow: []string{"test.>"}},
			},
		}
		repo.On("Get", "testuser").Return(testUser, true)

		arc := jwt.NewAuthorizationRequestClaims(userPubKey)
		arc.ConnectOptions.Username = "testuser"
		arc.ConnectOptions.Password = "password"
		arc.Server = jwt.ServerID{
			Name: "test-server",
			Host: "localhost",
			ID:   serverPubKey,
		}

		token, err := arc.Encode(issuerKP)
		require.NoError(t, err, "JWT encoding failed")

		req := &MockRequest{
			data:    []byte(token),
			headers: make(map[string][]string),
			subject: "test.subject",
		}
		req.On("Respond", mock.Anything, mock.Anything).Return(nil)

		handler.HandleRequest(req)

		repo.AssertExpectations(t)
		req.AssertCalled(t, "Respond", mock.Anything, mock.Anything)
	})
}
