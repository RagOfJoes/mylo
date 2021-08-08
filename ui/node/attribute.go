package node

// Attributes
//
type Attributes interface {
	ID() string
	Reset()
	GetValue() interface{}
	SetValue(value interface{})
}

// InputAttribute defines the structure for a normal
// Input Node
//
// Example: Username input
type InputAttribute struct {
	Name     string `json:"name" validate:"required"`
	Type     string `json:"type" validate:"required"`
	Label    string `json:"label"`
	Value    string `json:"value,omitempty"`
	Required bool   `json:"required"`
	Pattern  string `json:"pattern"`
	Disabled bool   `json:"disabled"`
}

// LinkAttribute defines the structure for a
// Link Node
//
// Example: Social Login Button
type LinkAttribute struct {
	Name  string `json:"name" validate:"required"`
	Label string `json:"label" validate:"required"`
	Value string `json:"link" validate:"required,url"`
}

// Implement NodeAttribute interface to
// Input, Link Attribute

func (i *InputAttribute) ID() string {
	return i.Name
}
func (i *InputAttribute) Reset() {
	i.Value = ""
}
func (i *InputAttribute) GetValue() interface{} {
	return i.Value
}
func (i *InputAttribute) SetValue(value interface{}) {
	i.Value, _ = value.(string)
}

func (l *LinkAttribute) ID() string {
	return ""
}
func (l *LinkAttribute) Reset() {}
func (l *LinkAttribute) GetValue() interface{} {
	return l.Value
}
func (l *LinkAttribute) SetValue(value interface{}) {
	l.Value, _ = value.(string)
}
