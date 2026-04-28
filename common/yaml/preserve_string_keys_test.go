package yaml_test

import (
	"testing"

	mihomoYaml "github.com/metacubex/mihomo/common/yaml"
	"github.com/stretchr/testify/require"
)

type stringPreserveConfig struct {
	ShortID string `yaml:"short-id"`
	Count   int    `yaml:"count"`
	Name    string `yaml:"name"`
	Nested  struct {
		ShortID []string `yaml:"short-id"`
		Value   int      `yaml:"value"`
	} `yaml:"nested"`
}

func TestUnmarshalPreservesShortIDAsString(t *testing.T) {
	input := []byte("short-id: 7266e6\ncount: 7\n")

	var cfg stringPreserveConfig
	require.NoError(t, mihomoYaml.Unmarshal(input, &cfg))
	require.Equal(t, "7266e6", cfg.ShortID)
	require.Equal(t, 7, cfg.Count)
}

func TestUnmarshalPreservesShortIDSequenceItemsAsString(t *testing.T) {
	input := []byte("nested:\n  short-id:\n    - 7266e6\n    - 10f897e26c4b9478\n  value: 3\n")

	var cfg stringPreserveConfig
	require.NoError(t, mihomoYaml.Unmarshal(input, &cfg))
	require.Equal(t, []string{"7266e6", "10f897e26c4b9478"}, cfg.Nested.ShortID)
	require.Equal(t, 3, cfg.Nested.Value)
}

func TestUnmarshalDoesNotAffectNonShortIDFields(t *testing.T) {
	input := []byte("count: 9\nname: example\n")

	var cfg stringPreserveConfig
	require.NoError(t, mihomoYaml.Unmarshal(input, &cfg))
	require.Equal(t, 9, cfg.Count)
	require.Equal(t, "example", cfg.Name)
}
