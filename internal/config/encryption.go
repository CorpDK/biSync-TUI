package config

// EncryptionConfig holds per-mapping encryption settings.
type EncryptionConfig struct {
	Enabled     bool   `toml:"enabled"`
	CryptRemote string `toml:"crypt_remote"`
}
