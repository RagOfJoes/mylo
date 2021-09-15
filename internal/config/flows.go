package config

import "time"

type Login struct {
	// URL for flow
	//
	// Default: login
	URL string
	// Lifetime of flow
	//
	// Default: 10m
	Lifetime time.Duration
}

type Registration struct {
	// URL for flow
	//
	// Default: login
	URL string
	// Lifetime of flow
	//
	// Default: 10m
	Lifetime time.Duration
}

type Verification struct {
	// URL for flow
	//
	// Default: login
	URL string
	// Lifetime of flow
	//
	// Default: 10m
	Lifetime time.Duration
}
