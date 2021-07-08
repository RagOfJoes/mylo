package service

import (
	"github.com/RagOfJoes/idp/ui/form"
	"github.com/RagOfJoes/idp/ui/node"
)

func generateForm(action string, csrfToken string) form.Form {
	return form.Form{
		Action: action,
		Method: form.POST,
		Nodes: node.Nodes{
			{
				Type:  node.Input,
				Group: node.Default,
				Attributes: &node.InputAttribute{
					Required:   true,
					Type:       "hidden",
					FieldValue: csrfToken,
					Name:       "csrf_token",
				},
			},
			{
				Type:  node.Input,
				Group: node.Password,
				Attributes: &node.InputAttribute{
					FieldValue: "",
					Required:   true,
					Type:       "text",
					Name:       "identifier",
					Label:      "Email or username",
				},
			},
			{
				Type:  node.Input,
				Group: node.Password,
				Attributes: &node.InputAttribute{
					FieldValue: "",
					Required:   true,
					Type:       "password",
					Name:       "password",
					Label:      "Password",
				},
			},
		},
	}
}
