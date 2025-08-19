package types

// DBConfig holds database-related configuration
type DBConfig struct {
	DBPath            string
	ActiveKeyVersion  int
	EncryptionKeys    map[int][]byte
	KeyDerivationSalt []byte
	EncryptionMode    string
	MasterKey         string
	EncryptionAlgo    string
	RetentionRuns     int
}

// GlobalConfig interface for configuration access
type GlobalConfig interface {
	GetDBConfig() *DBConfig
}
