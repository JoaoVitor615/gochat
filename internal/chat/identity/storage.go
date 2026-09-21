package identity

import (
	"os"
	"path/filepath"
)

func identityPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, MainDir, KeyFileName), nil
}

func StoreIdentity(identity []byte) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	identityDir := filepath.Join(homeDir, MainDir)
	if err := os.MkdirAll(identityDir, 0700); err != nil {
		return err
	}

	path := filepath.Join(identityDir, KeyFileName)
	if err := os.WriteFile(path, identity, 0600); err != nil {
		return err
	}

	// WriteFile does not change permissions when the file already exists.
	return os.Chmod(path, 0600)
}

// IdentityExists reports whether the identity file exists.
func IdentityExists() (bool, error) {
	path, err := identityPath()
	if err != nil {
		return false, err
	}

	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func LoadIdentity() ([]byte, error) {
	path, err := identityPath()
	if err != nil {
		return nil, err
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return contents, nil
}
