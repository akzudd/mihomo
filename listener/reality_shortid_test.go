package listener_test

import (
	"testing"

	mihomoYaml "github.com/metacubex/mihomo/common/yaml"
	inbound "github.com/metacubex/mihomo/listener/inbound"
	"github.com/metacubex/mihomo/listener"
	"github.com/stretchr/testify/require"
)

func TestParseListenerPreservesRealityShortIDFromYAML(t *testing.T) {
	input := []byte("name: reality-in\ntype: trojan\nlisten: 127.0.0.1\nport: 443\nusers:\n  - password: secret\nreality-config:\n  dest: example.com:443\n  private-key: YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWE\n  short-id:\n    - 7266e6\n  server-names:\n    - example.com\n")

	var mapping map[string]any
	require.NoError(t, mihomoYaml.Unmarshal(input, &mapping))

	inboundListener, err := listener.ParseListener(mapping)
	require.NoError(t, err)

	config, ok := inboundListener.Config().(*inbound.TrojanOption)
	require.True(t, ok)
	require.Equal(t, []string{"7266e6"}, config.RealityConfig.ShortID)
}
