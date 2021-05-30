package node

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
)

type Type string
type Group string

const (
	Link  Type = "link"
	Input Type = "input"

	OIDC     Group = "oidc"
	Password Group = "password"
)

type Node struct {
	Type       Type       `json:"type" validate:"required"`
	Group      Group      `json:"group" validate:"required,oneof='oidc' 'password'"`
	Attributes Attributes `json:"attributes" validate:"required"`
}

type Nodes []*Node

// Used for en/decoding the Attributes field.
type rawNode struct {
	Type       Type       `json:"type"`
	Group      Group      `json:"group"`
	Attributes Attributes `json:"attributes"`
}

func (n *Node) UnmarshalJSON(data []byte) error {
	// Cast Node's attribute to proper struct
	var attr Attributes
	t := gjson.GetBytes(data, "type").String()
	switch Type(t) {
	case Input:
		attr = new(InputAttribute)
	case Link:
		attr = new(LinkAttribute)
	default:
		return fmt.Errorf("unexpected node type: %s", t)
	}
	// Decode JSON to rawNode struct
	var d rawNode
	d.Attributes = attr
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&d); err != nil {
		return err
	}
	// Set Node to decoded JSON
	*n = Node(d)
	return nil
}

func (n *Node) MarshalJSON() ([]byte, error) {
	// Assign proper type to Node depending on
	// what type of Attribute was assigned to it
	var t Type
	if n.Attributes != nil {
		switch n.Attributes.(type) {
		case *InputAttribute:
			t = Input
		case *LinkAttribute:
			t = Link
		default:
			return nil, fmt.Errorf("unknown node type: %T", n.Attributes)
		}
	}

	// If no type is assigned to Node then use
	// what was found above
	if n.Type == "" {
		n.Type = t
	} else if n.Type != t {
		return nil, fmt.Errorf("node type and node attributes mismatch: %T != %s", n.Attributes, n.Type)
	}
	return json.Marshal((*rawNode)(n))
}
