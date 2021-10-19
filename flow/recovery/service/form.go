package service

import (
	"github.com/RagOfJoes/idp/ui/form"
	"github.com/RagOfJoes/idp/ui/node"
)

// Generate form for
func generateInitialForm(action string) form.Form {
	return form.Form{
		Action: action,
		Method: "POST",
		Nodes: node.Nodes{
			&node.Node{
				Type:  node.Input,
				Group: node.Default,
				Attributes: &node.InputAttribute{
					Required: true,
					Type:     "text",
					Name:     "identifier",
					Label:    "Identifier",
				},
			},
		},
	}
}

// Generate form for recovery
func generateRecoveryForm(action string) form.Form {
	return form.Form{
		Action: action,
		Method: "POST",
		Nodes: node.Nodes{
			&node.Node{
				Type:  node.Input,
				Group: node.Default,
				Attributes: &node.InputAttribute{
					Required: true,
					Name:     "password",
					Type:     "password",
					Label:    "New Password",
				},
			},
			&node.Node{
				Type:  node.Input,
				Group: node.Default,
				Attributes: &node.InputAttribute{
					Required: true,
					Type:     "password",
					Name:     "confirm_password",
					Label:    "Confirm New Password",
				},
			},
		},
	}
}
