package storage

import (
	"encoding/json"
	"fmt"
	"time"

	"set/internal/github"
	"set/internal/logger"

	bolt "go.etcd.io/bbolt"
)

const (
	// Bucket names
	issuesBucket   = "issues"
	prsBucket      = "prs"
	metadataBucket = "metadata"

	// Metadata keys
	lastSyncKey = "last_sync"
	repoKey     = "repository"
)

// Store handles data persistence using BoltDB
type Store struct {
	db   *bolt.DB
	path string
}

// NewStore creates a new storage instance
func NewStore(path string) (*Store, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create buckets if they don't exist
	err = db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range []string{issuesBucket, prsBucket, metadataBucket} {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	logger.Infof("Storage initialized at %s", path)
	return &Store{db: db, path: path}, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	if s.db != nil {
		logger.Info("Closing storage")
		return s.db.Close()
	}
	return nil
}

// SaveIssues saves a batch of issues to the database
func (s *Store) SaveIssues(issues []*github.Issue) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(issuesBucket))
		if bucket == nil {
			return fmt.Errorf("issues bucket not found")
		}

		for _, issue := range issues {
			key := []byte(fmt.Sprintf("%d", issue.Number))
			data, err := json.Marshal(issue)
			if err != nil {
				return fmt.Errorf("failed to marshal issue #%d: %w", issue.Number, err)
			}

			if err := bucket.Put(key, data); err != nil {
				return fmt.Errorf("failed to save issue #%d: %w", issue.Number, err)
			}
		}

		logger.Infof("Saved %d issues to storage", len(issues))
		return nil
	})
}

// SavePullRequests saves a batch of pull requests to the database
func (s *Store) SavePullRequests(prs []*github.PullRequest) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(prsBucket))
		if bucket == nil {
			return fmt.Errorf("prs bucket not found")
		}

		for _, pr := range prs {
			key := []byte(fmt.Sprintf("%d", pr.Number))
			data, err := json.Marshal(pr)
			if err != nil {
				return fmt.Errorf("failed to marshal PR #%d: %w", pr.Number, err)
			}

			if err := bucket.Put(key, data); err != nil {
				return fmt.Errorf("failed to save PR #%d: %w", pr.Number, err)
			}
		}

		logger.Infof("Saved %d pull requests to storage", len(prs))
		return nil
	})
}

// GetIssue retrieves a single issue by number
func (s *Store) GetIssue(number int) (*github.Issue, error) {
	var issue github.Issue

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(issuesBucket))
		if bucket == nil {
			return fmt.Errorf("issues bucket not found")
		}

		key := []byte(fmt.Sprintf("%d", number))
		data := bucket.Get(key)
		if data == nil {
			return fmt.Errorf("issue #%d not found", number)
		}

		return json.Unmarshal(data, &issue)
	})

	if err != nil {
		return nil, err
	}

	return &issue, nil
}

// GetPullRequest retrieves a single pull request by number
func (s *Store) GetPullRequest(number int) (*github.PullRequest, error) {
	var pr github.PullRequest

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(prsBucket))
		if bucket == nil {
			return fmt.Errorf("prs bucket not found")
		}

		key := []byte(fmt.Sprintf("%d", number))
		data := bucket.Get(key)
		if data == nil {
			return fmt.Errorf("pull request #%d not found", number)
		}

		return json.Unmarshal(data, &pr)
	})

	if err != nil {
		return nil, err
	}

	return &pr, nil
}

// GetAllIssues retrieves all issues from storage
func (s *Store) GetAllIssues() ([]*github.Issue, error) {
	var issues []*github.Issue

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(issuesBucket))
		if bucket == nil {
			return fmt.Errorf("issues bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var issue github.Issue
			if err := json.Unmarshal(v, &issue); err != nil {
				logger.Warnf("Failed to unmarshal issue: %v", err)
				return nil // Skip corrupted entries
			}
			issues = append(issues, &issue)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return issues, nil
}

// GetAllPullRequests retrieves all pull requests from storage
func (s *Store) GetAllPullRequests() ([]*github.PullRequest, error) {
	var prs []*github.PullRequest

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(prsBucket))
		if bucket == nil {
			return fmt.Errorf("prs bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var pr github.PullRequest
			if err := json.Unmarshal(v, &pr); err != nil {
				logger.Warnf("Failed to unmarshal pull request: %v", err)
				return nil // Skip corrupted entries
			}
			prs = append(prs, &pr)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return prs, nil
}

// SetLastSync records the last successful sync time
func (s *Store) SetLastSync(t time.Time) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(metadataBucket))
		if bucket == nil {
			return fmt.Errorf("metadata bucket not found")
		}

		data, err := t.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal time: %w", err)
		}

		return bucket.Put([]byte(lastSyncKey), data)
	})
}

// GetLastSync retrieves the last successful sync time
func (s *Store) GetLastSync() (*time.Time, error) {
	var syncTime time.Time

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(metadataBucket))
		if bucket == nil {
			return fmt.Errorf("metadata bucket not found")
		}

		data := bucket.Get([]byte(lastSyncKey))
		if data == nil {
			return nil // No sync yet
		}

		return syncTime.UnmarshalBinary(data)
	})

	if err != nil {
		return nil, err
	}

	// Return nil if no sync has occurred
	if syncTime.IsZero() {
		return nil, nil
	}

	return &syncTime, nil
}

// SetRepository stores repository information
func (s *Store) SetRepository(repo *github.Repository) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(metadataBucket))
		if bucket == nil {
			return fmt.Errorf("metadata bucket not found")
		}

		data, err := json.Marshal(repo)
		if err != nil {
			return fmt.Errorf("failed to marshal repository: %w", err)
		}

		return bucket.Put([]byte(repoKey), data)
	})
}

// GetRepository retrieves stored repository information
func (s *Store) GetRepository() (*github.Repository, error) {
	var repo github.Repository

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(metadataBucket))
		if bucket == nil {
			return fmt.Errorf("metadata bucket not found")
		}

		data := bucket.Get([]byte(repoKey))
		if data == nil {
			return fmt.Errorf("repository not found in storage")
		}

		return json.Unmarshal(data, &repo)
	})

	if err != nil {
		return nil, err
	}

	return &repo, nil
}

// CountIssues returns the number of issues in storage
func (s *Store) CountIssues() (int, error) {
	count := 0

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(issuesBucket))
		if bucket == nil {
			return fmt.Errorf("issues bucket not found")
		}

		count = bucket.Stats().KeyN
		return nil
	})

	return count, err
}

// CountPullRequests returns the number of pull requests in storage
func (s *Store) CountPullRequests() (int, error) {
	count := 0

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(prsBucket))
		if bucket == nil {
			return fmt.Errorf("prs bucket not found")
		}

		count = bucket.Stats().KeyN
		return nil
	})

	return count, err
}

// Clear removes all data from storage
func (s *Store) Clear() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		// Delete all buckets
		for _, bucketName := range []string{issuesBucket, prsBucket, metadataBucket} {
			if err := tx.DeleteBucket([]byte(bucketName)); err != nil && err != bolt.ErrBucketNotFound {
				return fmt.Errorf("failed to delete bucket %s: %w", bucketName, err)
			}
		}

		// Recreate buckets
		for _, bucketName := range []string{issuesBucket, prsBucket, metadataBucket} {
			if _, err := tx.CreateBucket([]byte(bucketName)); err != nil {
				return fmt.Errorf("failed to recreate bucket %s: %w", bucketName, err)
			}
		}

		logger.Info("Storage cleared")
		return nil
	})
}
