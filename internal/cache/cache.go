// ************************************************************************************************
// Package cache provides caching functionality using BadgerDB for the repomix-mcp application.
// It handles storage and retrieval of indexed repository content with efficient key-value operations
// and automatic expiration management.
package cache

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"repomix-mcp/pkg/types"

	"github.com/dgraph-io/badger/v4"
)

// ************************************************************************************************
// Cache manages BadgerDB storage for indexed repository content.
// It provides efficient storage and retrieval operations with automatic expiration
// and cache management capabilities.
type Cache struct {
	db     *badger.DB
	config *types.CacheConfig
}

// ************************************************************************************************
// NewCache creates a new cache instance with the specified configuration.
// It initializes the BadgerDB database and prepares it for storage operations.
//
// Returns:
//   - *Cache: The cache instance.
//   - error: An error if cache initialization fails.
//
// Example usage:
//
//	cache, err := NewCache(&config.Cache)
//	if err != nil {
//		return fmt.Errorf("failed to create cache: %w", err)
//	}
//	defer cache.Close()
func NewCache(config *types.CacheConfig) (*Cache, error) {
	if config == nil {
		return nil, fmt.Errorf("%w: cache config is nil", types.ErrInvalidConfig)
	}

	// Ensure cache directory exists
	if err := mock_osMkdirAll(config.Path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory\n>    %w", err)
	}

	// Configure BadgerDB options
	opts := badger.DefaultOptions(config.Path)
	opts.Logger = nil // Disable BadgerDB logging

	// Open BadgerDB
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to open BadgerDB\n>    %w", types.ErrCacheInitFailed, err)
	}

	cache := &Cache{
		db:     db,
		config: config,
	}

	return cache, nil
}

// ************************************************************************************************
// Close closes the cache database connection.
// This method should be called when shutting down the application.
//
// Returns:
//   - error: An error if closing fails.
//
// Example usage:
//
//	defer cache.Close()
func (c *Cache) Close() error {
	if c.db == nil {
		return nil
	}
	
	if err := c.db.Close(); err != nil {
		return fmt.Errorf("failed to close cache database\n>    %w", err)
	}
	
	return nil
}

// ************************************************************************************************
// StoreRepository stores a complete repository index in the cache.
// It serializes the repository data and stores it with an expiration time.
//
// Returns:
//   - error: An error if storage fails.
//
// Example usage:
//
//	err := cache.StoreRepository(&repositoryIndex)
//	if err != nil {
//		return fmt.Errorf("failed to store repository: %w", err)
//	}
func (c *Cache) StoreRepository(repo *types.RepositoryIndex) error {
	if repo == nil {
		return fmt.Errorf("%w: repository index is nil", types.ErrInvalidConfig)
	}

	// Serialize repository data
	data, err := json.Marshal(repo)
	if err != nil {
		return fmt.Errorf("failed to marshal repository data\n>    %w", err)
	}

	// Create cache key
	key := fmt.Sprintf("repo:%s", repo.ID)

	// Store in BadgerDB with TTL
	return c.db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(key), data)
		
		// Set TTL if configured
		if c.config.TTL != "" {
			ttl, err := mock_timeParseDuration(c.config.TTL)
			if err == nil {
				entry = entry.WithTTL(ttl)
			}
		}
		
		return txn.SetEntry(entry)
	})
}

// ************************************************************************************************
// GetRepository retrieves a repository index from the cache.
// It deserializes the stored data and returns the repository information.
//
// Returns:
//   - *types.RepositoryIndex: The repository index if found.
//   - error: An error if retrieval fails or repository is not found.
//
// Example usage:
//
//	repo, err := cache.GetRepository("my-repo")
//	if err != nil {
//		return fmt.Errorf("repository not found: %w", err)
//	}
func (c *Cache) GetRepository(repositoryID string) (*types.RepositoryIndex, error) {
	if repositoryID == "" {
		return nil, fmt.Errorf("%w: repository ID is empty", types.ErrInvalidConfig)
	}

	key := fmt.Sprintf("repo:%s", repositoryID)
	var repoData []byte

	err := c.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			repoData = append([]byte{}, val...)
			return nil
		})
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, fmt.Errorf("%w: %s", types.ErrRepositoryNotFound, repositoryID)
		}
		return nil, fmt.Errorf("failed to get repository from cache\n>    %w", err)
	}

	// Deserialize repository data
	var repo types.RepositoryIndex
	if err := json.Unmarshal(repoData, &repo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repository data\n>    %w", err)
	}

	return &repo, nil
}

// ************************************************************************************************
// StoreFile stores an individual file in the cache.
// It creates a separate cache entry for the file to enable efficient file-level operations.
//
// Returns:
//   - error: An error if storage fails.
//
// Example usage:
//
//	err := cache.StoreFile("my-repo", &indexedFile)
//	if err != nil {
//		return fmt.Errorf("failed to store file: %w", err)
//	}
func (c *Cache) StoreFile(repositoryID string, file *types.IndexedFile) error {
	if repositoryID == "" || file == nil {
		return fmt.Errorf("%w: invalid parameters", types.ErrInvalidConfig)
	}

	// Serialize file data
	data, err := json.Marshal(file)
	if err != nil {
		return fmt.Errorf("failed to marshal file data\n>    %w", err)
	}

	// Create cache key
	key := fmt.Sprintf("file:%s:%s", repositoryID, file.Path)

	// Store in BadgerDB with TTL
	return c.db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(key), data)
		
		// Set TTL if configured
		if c.config.TTL != "" {
			ttl, err := mock_timeParseDuration(c.config.TTL)
			if err == nil {
				entry = entry.WithTTL(ttl)
			}
		}
		
		return txn.SetEntry(entry)
	})
}

// ************************************************************************************************
// GetFile retrieves a specific file from the cache.
// It looks up the file by repository ID and file path.
//
// Returns:
//   - *types.IndexedFile: The indexed file if found.
//   - error: An error if retrieval fails or file is not found.
//
// Example usage:
//
//	file, err := cache.GetFile("my-repo", "src/main.go")
//	if err != nil {
//		return fmt.Errorf("file not found: %w", err)
//	}
func (c *Cache) GetFile(repositoryID, filePath string) (*types.IndexedFile, error) {
	if repositoryID == "" || filePath == "" {
		return nil, fmt.Errorf("%w: invalid parameters", types.ErrInvalidConfig)
	}

	key := fmt.Sprintf("file:%s:%s", repositoryID, filePath)
	var fileData []byte

	err := c.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			fileData = append([]byte{}, val...)
			return nil
		})
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, fmt.Errorf("%w: %s", types.ErrFileNotFound, filePath)
		}
		return nil, fmt.Errorf("failed to get file from cache\n>    %w", err)
	}

	// Deserialize file data
	var file types.IndexedFile
	if err := json.Unmarshal(fileData, &file); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file data\n>    %w", err)
	}

	return &file, nil
}

// ************************************************************************************************
// ListRepositories returns all cached repository IDs.
// It scans the cache for repository entries and returns their identifiers.
//
// Returns:
//   - []string: List of repository IDs.
//   - error: An error if scanning fails.
//
// Example usage:
//
//	repos, err := cache.ListRepositories()
//	if err != nil {
//		return fmt.Errorf("failed to list repositories: %w", err)
//	}
func (c *Cache) ListRepositories() ([]string, error) {
	var repositories []string

	err := c.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("repo:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := string(item.Key())
			
			// Extract repository ID from key (remove "repo:" prefix)
			if len(key) > 5 {
				repoID := key[5:]
				repositories = append(repositories, repoID)
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list repositories\n>    %w", err)
	}

	return repositories, nil
}

// ************************************************************************************************
// DeleteRepository removes a repository and all its associated files from the cache.
// It performs a cascading delete operation to maintain cache consistency.
//
// Returns:
//   - error: An error if deletion fails.
//
// Example usage:
//
//	err := cache.DeleteRepository("my-repo")
//	if err != nil {
//		return fmt.Errorf("failed to delete repository: %w", err)
//	}
func (c *Cache) DeleteRepository(repositoryID string) error {
	if repositoryID == "" {
		return fmt.Errorf("%w: repository ID is empty", types.ErrInvalidConfig)
	}

	return c.db.Update(func(txn *badger.Txn) error {
		// Delete repository entry
		repoKey := fmt.Sprintf("repo:%s", repositoryID)
		if err := txn.Delete([]byte(repoKey)); err != nil && err != badger.ErrKeyNotFound {
			return fmt.Errorf("failed to delete repository entry\n>    %w", err)
		}

		// Delete all associated files
		filePrefix := fmt.Sprintf("file:%s:", repositoryID)
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		var keysToDelete [][]byte
		for it.Seek([]byte(filePrefix)); it.ValidForPrefix([]byte(filePrefix)); it.Next() {
			item := it.Item()
			key := item.KeyCopy(nil)
			keysToDelete = append(keysToDelete, key)
		}

		// Delete collected keys
		for _, key := range keysToDelete {
			if err := txn.Delete(key); err != nil {
				return fmt.Errorf("failed to delete file entry\n>    %w", err)
			}
		}

		return nil
	})
}

// ************************************************************************************************
// GetCacheStats returns statistics about the cache usage.
// It provides information about storage usage and entry counts.
//
// Returns:
//   - map[string]interface{}: Cache statistics.
//   - error: An error if stats collection fails.
//
// Example usage:
//
//	stats, err := cache.GetCacheStats()
//	if err != nil {
//		return fmt.Errorf("failed to get cache stats: %w", err)
//	}
func (c *Cache) GetCacheStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Get BadgerDB statistics
	lsm, vlog := c.db.Size()
	stats["lsm_size"] = lsm
	stats["vlog_size"] = vlog
	stats["total_size"] = lsm + vlog
	stats["cache_path"] = c.config.Path

	// Count entries
	repoCount := 0
	fileCount := 0

	err := c.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())
			
			if filepath.HasPrefix(key, "repo:") {
				repoCount++
			} else if filepath.HasPrefix(key, "file:") {
				fileCount++
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to collect cache statistics\n>    %w", err)
	}

	stats["repository_count"] = repoCount
	stats["file_count"] = fileCount
	
	return stats, nil
}

// ************************************************************************************************
// RunGarbageCollection performs garbage collection on the cache database.
// It removes expired entries and optimizes storage usage.
//
// Returns:
//   - error: An error if garbage collection fails.
//
// Example usage:
//
//	err := cache.RunGarbageCollection()
//	if err != nil {
//		return fmt.Errorf("garbage collection failed: %w", err)
//	}
func (c *Cache) RunGarbageCollection() error {
	return c.db.RunValueLogGC(0.5)
}