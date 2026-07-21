package credentials

import (
	"fmt"
	"os"
	"path/filepath"
)

// Update applies mutate to the credentials file under an exclusive
// interprocess lock: it acquires the lock, reloads the current contents,
// applies the mutation, and saves. This makes concurrent CLI processes
// (login, logout, token refresh) serialize their read-modify-write cycles
// instead of overwriting each other's changes with stale snapshots.
func Update(path string, mutate func(*File) error) (*File, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("failed to create credentials directory: %w", err)
		}
	}

	// The lock lives in a sidecar file so locking never interferes with the
	// atomic rename in Save.
	// #nosec G304 -- sidecar of the user's own credentials file
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open credentials lock file: %w", err)
	}
	defer func() { _ = lock.Close() }()

	if err := lockFile(lock); err != nil {
		return nil, fmt.Errorf("failed to lock credentials file: %w", err)
	}
	defer func() { _ = unlockFile(lock) }()

	f, err := Load(path)
	if err != nil {
		return nil, err
	}
	if err := mutate(f); err != nil {
		return nil, err
	}
	if err := Save(path, f); err != nil {
		return nil, err
	}
	return f, nil
}
