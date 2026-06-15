package engine

type WriteMode string

const (
	WriteModePreview WriteMode = "preview"
	WriteModeApply   WriteMode = "apply"
	WriteModeStage   WriteMode = "stage"
)

type Config struct {
	AllowedRoots       []string
	TransactionLogDir  string
	WriteMode          WriteMode
	EnableStaging      bool
	StagingDatabaseURI string
	StageDir           string
}
