package service

import (
	"github.com/RagOfJoes/idp/ui/form"
	"github.com/RagOfJoes/idp/ui/node"
)

// Generates form for soft login
func generatePasswordForm(action string) form.Form {
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
					Label:    "Password",
				},
			},
		},
	}
}
