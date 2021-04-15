package internal

import "fmt"

var commitID string
var version string

const (
	EnvProduction  = "prod"
	EnvDevelopment = "dev"
)

func GetVersion() string {
	return version
}

func GetCommitID() string {
	return commitID
}

func GetBuildInfo() string {
	return fmt.Sprintf("CommitID: %s\nVersion: %s\nEnv: %s\n", commitID, version, environment)
}

func IsDev() bool {
	return environment == EnvDevelopment
}

func GetEnv() string {
	return environment
}
