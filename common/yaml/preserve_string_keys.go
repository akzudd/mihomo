package yaml

import "gopkg.in/yaml.v3"

const preservedStringKey = "short-id"
const yamlStringTag = "!!str"

func preserveStringKeyNodes(node *yaml.Node) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, child := range node.Content {
			preserveStringKeyNodes(child)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			if keyNode != nil && keyNode.Value == preservedStringKey {
				preserveStringValueNode(valueNode)
			}
			preserveStringKeyNodes(valueNode)
		}
	}
}

func preserveStringValueNode(node *yaml.Node) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.ScalarNode:
		node.Tag = yamlStringTag
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if child != nil && child.Kind == yaml.ScalarNode {
				child.Tag = yamlStringTag
			}
		}
	}
}
