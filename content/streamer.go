package content

import (
	"encoding/hex"
	"errors"
	"path/filepath"
)

var (
	VaultFilenameEmptyErr = errors.New("vault filename required but was empty")

	vaultKey = "key"
)

//go:generate mockgen -destination mocks/mocks.go -package mocks github.com/ONSdigital/dp-download-service/content VaultClient

// VaultClient is an interface to represent methods called to action upon vault
type VaultClient interface {
	ReadKey(path, key string) (string, error)
}

type Streamer struct {
	VaultCli  VaultClient
	VaultPath string
}

func New(vc VaultClient, vp string) *Streamer {
	return &Streamer{VaultCli: vc, VaultPath: vp}
}

func (s *Streamer) getVaultKeyForFile(filename string) ([]byte, error) {
	if len(filename) == 0 {
		return nil, VaultFilenameEmptyErr
	}

	vp := s.VaultPath + "/" + filepath.Base(filename)
	pskStr, err := s.VaultCli.ReadKey(vp, vaultKey)
	if err != nil {
		return nil, err
	}

	psk, err := hex.DecodeString(pskStr)
	if err != nil {
		return nil, err
	}

	return psk, nil
}
