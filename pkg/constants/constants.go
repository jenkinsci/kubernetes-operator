package constants

const (
	// OperatorName is a operator name
	OperatorName = "jenkins-operator"
	// DefaultAmountOfExecutors is the default amount of Jenkins executors
	DefaultAmountOfExecutors = 0
	// SeedJobSuffix is a suffix added for all seed jobs
	SeedJobSuffix = "job-dsl-seed"
	// DefaultJenkinsMasterImage is the default Jenkins master docker image
	DefaultJenkinsMasterImage = "jenkins/jenkins:2.319.3-lts"
	// DefaultHTTPPortInt32 is the default Jenkins HTTP port
	DefaultHTTPPortInt32 = int32(8080)
	// DefaultSlavePortInt32 is the default Jenkins port for slaves
	DefaultSlavePortInt32 = int32(50000)
	// JavaOpsVariableName is the name of environment variable which consists Jenkins Java options
	JavaOpsVariableName = "JAVA_OPTS"
)
