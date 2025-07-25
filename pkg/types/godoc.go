// ************************************************************************************************
// Package types provides Go module documentation related data structures for the repomix-mcp application.
// This file contains types specific to Go module documentation retrieval and configuration.
package types

// ************************************************************************************************
// GoModuleConfig defines configuration options for Go module documentation retrieval.
// It controls how Go modules are fetched, cached, and processed.
type GoModuleConfig struct {
	Enabled        bool   `json:"enabled" mapstructure:"enabled"`               // Whether Go module fallback is enabled
	TempDirBase    string `json:"tempDirBase" mapstructure:"tempDirBase"`       // Base directory for temporary Go modules
	CacheTimeout   string `json:"cacheTimeout" mapstructure:"cacheTimeout"`     // How long to cache Go module docs
	CommandTimeout string `json:"commandTimeout" mapstructure:"commandTimeout"` // Timeout for individual Go commands
	MaxRetries     int    `json:"maxRetries" mapstructure:"maxRetries"`         // Maximum retries for failed commands
	MaxConcurrent  int    `json:"maxConcurrent" mapstructure:"maxConcurrent"`   // Maximum concurrent Go operations
}