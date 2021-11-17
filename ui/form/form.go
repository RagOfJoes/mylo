package form

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/RagOfJoes/mylo/internal/validate"
	"github.com/RagOfJoes/mylo/ui/node"
)

const (
	GET  Method = "GET"
	PUT  Method = "PUT"
	POST Method = "POST"
)

type Form struct {
	Action string     `json:"action" validate:"required"`
	Method Method     `json:"method" validate:"required,oneof='GET' 'POST' 'PUT'"`
	Nodes  node.Nodes `json:"nodes" validate:"required"`
}

type Method string

// GORM custom data type funcs for Scanner and Valuer
// interfaces

// Value returns stringified version of JSON
func (f *Form) Value() (driver.Value, error) {
	if f == nil {
		return nil, nil
	}
	if err := validate.Check(f); err != nil {
		return nil, err
	}
	for _, n := range f.Nodes {
		if err := validate.Check(n); err != nil {
			return nil, err
		}
	}
	val, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	return string(val), nil
}

// Scan scans value into Form struct
func (f *Form) Scan(src interface{}) error {
	var bytes []byte
	switch v := src.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("failed to unmarshal JSON value: %T", src)
	}
	// Decode stringified JSON to Form
	var dest Form
	err := json.Unmarshal(bytes, &dest)
	*f = dest
	return err
}
