package servers

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildOIDCURI_AddsBothParamsWhenAbsent(t *testing.T) {
	in := "mongodb://example.com:27017/?retryWrites=true"
	out, err := buildOIDCURI(in)
	require.NoError(t, err)

	u, err := url.Parse(out)
	require.NoError(t, err)
	q := u.Query()
	assert.Equal(t, "MONGODB-OIDC", q.Get("authMechanism"))
	assert.Equal(t, "$external", q.Get("authSource"))
	assert.Equal(t, "true", q.Get("retryWrites"))
}

// ALLOWED_HOSTS is a client-side-only OIDC property that Compass rejects, so it
// must never appear in the copied URI. authSource must be encoded as
// %24external (Compass rejects a bare "$external").
func TestBuildOIDCURI_OmitsAllowedHostsAndEncodesExternal(t *testing.T) {
	in := "mongodb://example.com:27017/"
	out, err := buildOIDCURI(in)
	require.NoError(t, err)

	assert.NotContains(t, out, "ALLOWED_HOSTS")
	assert.NotContains(t, out, "authMechanismProperties")
	assert.Contains(t, out, "authSource=%24external")
}

func TestBuildOIDCURI_PreservesUserMechProps(t *testing.T) {
	in := "mongodb://example.com/?authMechanism=MONGODB-OIDC&authMechanismProperties=ENVIRONMENT:azure"
	out, err := buildOIDCURI(in)
	require.NoError(t, err)

	u, err := url.Parse(out)
	require.NoError(t, err)
	q := u.Query()
	assert.Equal(t, "MONGODB-OIDC", q.Get("authMechanism"))
	assert.Equal(t, "ENVIRONMENT:azure", q.Get("authMechanismProperties"))
	assert.Equal(t, "$external", q.Get("authSource"))
}

func TestBuildOIDCURI_PreservesExistingAuthSource(t *testing.T) {
	in := "mongodb://example.com/?authMechanism=MONGODB-OIDC&authSource=admin"
	out, err := buildOIDCURI(in)
	require.NoError(t, err)

	u, err := url.Parse(out)
	require.NoError(t, err)
	assert.Equal(t, "admin", u.Query().Get("authSource"))
}

func TestBuildOIDCURI_FillsParamsWhenMechanismPresent(t *testing.T) {
	in := "mongodb://example.com/?authMechanism=MONGODB-OIDC"
	out, err := buildOIDCURI(in)
	require.NoError(t, err)

	u, err := url.Parse(out)
	require.NoError(t, err)
	q := u.Query()
	assert.Equal(t, "MONGODB-OIDC", q.Get("authMechanism"))
	assert.Equal(t, "$external", q.Get("authSource"))
}

func TestBuildOIDCURI_PreservesUserInfoAndHostList(t *testing.T) {
	in := "mongodb://user:pass@host1:27017,host2:27017/admin?ssl=true"
	out, err := buildOIDCURI(in)
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(out, "mongodb://user:pass@host1:27017,host2:27017/admin?"),
		"expected user info, host list and path preserved, got %q", out)

	u, err := url.Parse(out)
	require.NoError(t, err)
	q := u.Query()
	assert.Equal(t, "true", q.Get("ssl"))
	assert.Equal(t, "MONGODB-OIDC", q.Get("authMechanism"))
	assert.Equal(t, "$external", q.Get("authSource"))
}

func TestBuildOIDCURI_SrvScheme(t *testing.T) {
	in := "mongodb+srv://cluster.example.com/"
	out, err := buildOIDCURI(in)
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(out, "mongodb+srv://cluster.example.com/"),
		"expected scheme preserved, got %q", out)
	u, err := url.Parse(out)
	require.NoError(t, err)
	assert.Equal(t, "MONGODB-OIDC", u.Query().Get("authMechanism"))
	assert.Equal(t, "$external", u.Query().Get("authSource"))
}

func TestBuildOIDCURI_ErrorOnUnparseable(t *testing.T) {
	_, err := buildOIDCURI("://not a url")
	assert.Error(t, err)
}
