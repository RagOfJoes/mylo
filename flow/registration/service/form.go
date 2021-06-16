package service

import (
	"github.com/RagOfJoes/idp/ui/form"
	"github.com/RagOfJoes/idp/ui/node"
)

func generateForm(action string, csrfToken string) form.Form {
	f := form.Form{
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
					Label:      "Username",
					Name:       "username",
				},
			},
			{
				Type:  node.Input,
				Group: node.Password,
				Attributes: &node.InputAttribute{
					FieldValue: "",
					Required:   true,
					Type:       "email",
					Name:       "email",
					Label:      "Email Address",
				},
			},
			{
				Type:  node.Input,
				Group: node.Password,
				Attributes: &node.InputAttribute{
					FieldValue: "",
					Type:       "password",
					Name:       "password",
					Label:      "Password",
				},
			},
			{
				Type:  node.Input,
				Group: node.Password,
				Attributes: &node.InputAttribute{
					FieldValue: "",
					Type:       "text",
					Name:       "first_name",
					Label:      "First Name",
				},
			},
			{
				Type:  node.Input,
				Group: node.Password,
				Attributes: &node.InputAttribute{
					FieldValue: "",
					Type:       "text",
					Name:       "last_name",
					Label:      "Last Name",
				},
			},
		},
	}

	return f
}
