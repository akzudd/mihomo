// Package yaml provides a common entrance for YAML marshaling and unmarshalling.
package yaml

import (
	"gopkg.in/yaml.v3"
)

func Unmarshal(in []byte, out any) (err error) {
	var node yaml.Node
	if err = yaml.Unmarshal(in, &node); err != nil {
		return err
	}
	preserveStringKeyNodes(&node)
	return node.Decode(out)
}

func Marshal(in any) (out []byte, err error) {
	return yaml.Marshal(in)
}
