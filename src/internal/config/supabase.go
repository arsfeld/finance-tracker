package config

import (
	"fmt"
	"os"

	supa "github.com/supabase-community/supabase-go"
)

type SupabaseConfig struct {
	URL        string
	AnonKey    string
	ServiceKey string
}

// Client wraps the Supabase client with both anon and service clients
type Client struct {
	Anon    *supa.Client // For client-side operations
	Service *supa.Client // For server-side operations with full access
}

// NewSupabaseClient creates both anon and service Supabase clients
func NewSupabaseClient() (*Client, error) {
	config := &SupabaseConfig{
		URL:        os.Getenv("SUPABASE_URL"),
		AnonKey:    os.Getenv("SUPABASE_ANON_KEY"),
		ServiceKey: os.Getenv("SUPABASE_SERVICE_KEY"),
	}

	if config.URL == "" || config.AnonKey == "" {
		return nil, fmt.Errorf("SUPABASE_URL and SUPABASE_ANON_KEY are required")
	}

	// Create anon client for regular operations
	anonClient, err := supa.NewClient(config.URL, config.AnonKey, &supa.ClientOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create anon client: %w", err)
	}

	// Create service client for admin operations if service key is provided
	var serviceClient *supa.Client
	if config.ServiceKey != "" {
		serviceClient, err = supa.NewClient(config.URL, config.ServiceKey, &supa.ClientOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create service client: %w", err)
		}
	}

	return &Client{
		Anon:    anonClient,
		Service: serviceClient,
	}, nil
}