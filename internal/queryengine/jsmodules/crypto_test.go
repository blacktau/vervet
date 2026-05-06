package jsmodules

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrypto_Sha256Hex(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('crypto').createHash('sha256').update('abc').digest('hex')`)
	require.NoError(t, err)
	assert.Equal(t,
		"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		val.Export(),
	)
}

func TestCrypto_Md5Hex(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('crypto').createHash('md5').update('abc').digest('hex')`)
	require.NoError(t, err)
	assert.Equal(t, "900150983cd24fb0d6963f7d28e17f72", val.Export())
}

func TestCrypto_Sha1Base64(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('crypto').createHash('sha1').update('abc').digest('base64')`)
	require.NoError(t, err)
	assert.Equal(t, "qZk+NkcGgWq6PiVxeFDCbJzQ2J0=", val.Export())
}

func TestCrypto_HashChainedUpdates(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`
		const h = require('crypto').createHash('sha256');
		h.update('a'); h.update('b'); h.update('c');
		h.digest('hex')
	`)
	require.NoError(t, err)
	assert.Equal(t,
		"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		val.Export(),
	)
}

func TestCrypto_UnknownAlgPanics(t *testing.T) {
	rt := newTestRuntime(t)
	_, err := rt.RunString(`require('crypto').createHash('nope')`)
	require.Error(t, err)
}

func TestCrypto_RandomUUID(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('crypto').randomUUID()`)
	require.NoError(t, err)
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	assert.Regexp(t, re, val.Export())
}

func TestCrypto_RandomBytesLength(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('crypto').randomBytes(16).byteLength`)
	require.NoError(t, err)
	assert.EqualValues(t, 16, val.Export())
}
