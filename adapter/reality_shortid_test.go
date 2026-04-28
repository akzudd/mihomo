package adapter_test

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/metacubex/mihomo/adapter"
	mihomoYaml "github.com/metacubex/mihomo/common/yaml"
	"github.com/stretchr/testify/require"
)

func TestParseProxyPreservesRealityShortIDFromYAML(t *testing.T) {
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	publicKey := base64.RawURLEncoding.EncodeToString(privateKey.PublicKey().Bytes())

	input := []byte("name: reality-test\ntype: trojan\nserver: example.com\nport: 443\npassword: secret\nreality-opts:\n  public-key: " + publicKey + "\n  short-id: 7266e6\n")

	var mapping map[string]any
	require.NoError(t, mihomoYaml.Unmarshal(input, &mapping))

	_, err = adapter.ParseProxy(mapping)
	require.NoError(t, err)
}
