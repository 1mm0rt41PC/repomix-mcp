package types

import "fmt"

var (
	ErrConfigNotFound        = fmt.Errorf("0x%X%X config_not_found", "REPOMIX", []byte{0x01})
	ErrInvalidConfig         = fmt.Errorf("0x%X%X invalid_config", "REPOMIX", []byte{0x02})
	ErrRepositoryNotFound    = fmt.Errorf("0x%X%X repository_not_found", "REPOMIX", []byte{0x03})
	ErrInvalidRepositoryType = fmt.Errorf("0x%X%X invalid_repository_type", "REPOMIX", []byte{0x04})
	ErrAuthenticationFailed  = fmt.Errorf("0x%X%X authentication_failed", "REPOMIX", []byte{0x05})
	ErrCacheInitFailed       = fmt.Errorf("0x%X%X cache_init_failed", "REPOMIX", []byte{0x06})
	ErrCacheCorrupted        = fmt.Errorf("0x%X%X cache_corrupted", "REPOMIX", []byte{0x07})
	ErrIndexingFailed        = fmt.Errorf("0x%X%X indexing_failed", "REPOMIX", []byte{0x08})
	ErrRepomixNotFound       = fmt.Errorf("0x%X%X repomix_not_found", "REPOMIX", []byte{0x09})
	ErrRepomixExecFailed     = fmt.Errorf("0x%X%X repomix_exec_failed", "REPOMIX", []byte{0x0A})
	ErrGitCloneFailed        = fmt.Errorf("0x%X%X git_clone_failed", "REPOMIX", []byte{0x0B})
	ErrGitPullFailed         = fmt.Errorf("0x%X%X git_pull_failed", "REPOMIX", []byte{0x0C})
	ErrInvalidSearchQuery    = fmt.Errorf("0x%X%X invalid_search_query", "REPOMIX", []byte{0x0D})
	ErrSearchFailed          = fmt.Errorf("0x%X%X search_failed", "REPOMIX", []byte{0x0E})
	ErrMCPRequestInvalid     = fmt.Errorf("0x%X%X mcp_request_invalid", "REPOMIX", []byte{0x0F})
	ErrMCPToolNotFound       = fmt.Errorf("0x%X%X mcp_tool_not_found", "REPOMIX", []byte{0x10})
	ErrFileNotFound          = fmt.Errorf("0x%X%X file_not_found", "REPOMIX", []byte{0x11})
	ErrPermissionDenied      = fmt.Errorf("0x%X%X permission_denied", "REPOMIX", []byte{0x12})
	ErrInvalidPath           = fmt.Errorf("0x%X%X invalid_path", "REPOMIX", []byte{0x13})
	ErrNetworkError          = fmt.Errorf("0x%X%X network_error", "REPOMIX", []byte{0x14})
	ErrTimeoutError          = fmt.Errorf("0x%X%X timeout_error", "REPOMIX", []byte{0x15})
	ErrNotInitialized        = fmt.Errorf("0x%X%X not_initialized", "REPOMIX", []byte{0x16})
	ErrConcurrentAccess      = fmt.Errorf("0x%X%X concurrent_access", "REPOMIX", []byte{0x17})
)