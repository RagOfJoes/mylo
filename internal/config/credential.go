package config

type Argon struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

type Credential struct {
	MinimumScore int `validate:"min=0,max=4"`
	Argon        Argon
}
