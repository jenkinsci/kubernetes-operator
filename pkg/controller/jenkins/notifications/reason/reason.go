package reason

const (
	// OperatorSource defines that notification concerns operator
	OperatorSource Source = "operator"
	// KubernetesSource defines that notification concerns kubernetes
	KubernetesSource Source = "kubernetes"
	// HumanSource defines that notification concerns human
	HumanSource Source = "human"
)

// Reason is interface that let us know why operator sent notification
type Reason interface {
	Short() []string
	Verbose() []string
}

// Undefined is base or untraceable reason
type Undefined struct {
	source  Source
	short   []string
	verbose []string
}

// PodRestart defines the reason why Jenkins master pod restarted
type PodRestart struct {
	Undefined
}

// PodCreation informs that pod is being created
type PodCreation struct {
	Undefined
}

// ReconcileLoopFailed defines the reason why the reconcile loop failed
type ReconcileLoopFailed struct {
	Undefined
}

// GroovyScriptExecutionFailed defines the reason why the groovy script execution failed
type GroovyScriptExecutionFailed struct {
	Undefined
}

// BaseConfigurationFailed defines the reason why base configuration phase failed
type BaseConfigurationFailed struct {
	Undefined
}

// BaseConfigurationComplete informs that base configuration is valid and complete
type BaseConfigurationComplete struct {
	Undefined
}

// UserConfigurationFailed defines the reason why user configuration phase failed
type UserConfigurationFailed struct {
	Undefined
}

// UserConfigurationComplete informs that user configuration is valid and complete
type UserConfigurationComplete struct {
	Undefined
}

// NewUndefined returns new instance of Undefined
func NewUndefined(source Source, short []string, verbose ...string) *Undefined {
	copyToVerboseIfNil(short, &verbose)
	return &Undefined{source: source, short: short, verbose: verbose}
}

// NewPodRestart returns new instance of PodRestart
func NewPodRestart(source Source, short []string, verbose ...string) *PodRestart {
	copyToVerboseIfNil(short, &verbose)
	return &PodRestart{Undefined{
		source:  source,
		short:   short,
		verbose: verbose,
	}}
}

// NewPodCreation returns new instance of PodCreation
func NewPodCreation(source Source, short []string, verbose ...string) *PodCreation {
	copyToVerboseIfNil(short, &verbose)
	return &PodCreation{Undefined{
		source:  source,
		short:   short,
		verbose: verbose,
	}}
}

// NewReconcileLoopFailed returns new instance of ReconcileLoopFailed
func NewReconcileLoopFailed(source Source, short []string, verbose ...string) *ReconcileLoopFailed {
	copyToVerboseIfNil(short, &verbose)
	return &ReconcileLoopFailed{Undefined{
		source:  source,
		short:   short,
		verbose: verbose,
	}}
}

// NewGroovyScriptExecutionFailed returns new instance of GroovyScriptExecutionFailed
func NewGroovyScriptExecutionFailed(source Source, short []string, verbose ...string) *GroovyScriptExecutionFailed {
	copyToVerboseIfNil(short, &verbose)
	return &GroovyScriptExecutionFailed{Undefined{
		source:  source,
		short:   short,
		verbose: verbose,
	}}
}

// NewBaseConfigurationFailed returns new instance of BaseConfigurationFailed
func NewBaseConfigurationFailed(source Source, short []string, verbose ...string) *BaseConfigurationFailed {
	copyToVerboseIfNil(short, &verbose)
	return &BaseConfigurationFailed{Undefined{
		source:  source,
		short:   short,
		verbose: verbose,
	}}
}

// NewBaseConfigurationComplete returns new instance of BaseConfigurationComplete
func NewBaseConfigurationComplete(source Source, short []string, verbose ...string) *BaseConfigurationComplete {
	copyToVerboseIfNil(short, &verbose)
	return &BaseConfigurationComplete{Undefined{
		source:  source,
		short:   short,
		verbose: verbose,
	}}
}

// NewUserConfigurationFailed returns new instance of UserConfigurationFailed
func NewUserConfigurationFailed(source Source, short []string, verbose ...string) *UserConfigurationFailed {
	copyToVerboseIfNil(short, &verbose)
	return &UserConfigurationFailed{Undefined{
		source:  source,
		short:   short,
		verbose: verbose,
	}}
}

// NewUserConfigurationComplete returns new instance of UserConfigurationComplete
func NewUserConfigurationComplete(source Source, short []string, verbose ...string) *UserConfigurationComplete {
	copyToVerboseIfNil(short, &verbose)
	return &UserConfigurationComplete{Undefined{
		source:  source,
		short:   short,
		verbose: verbose,
	}}
}

// Source is enum type that informs us what triggered notification
type Source string

// Short is list of reasons
func (p Undefined) Short() []string {
	return p.short
}

// Verbose is list of reasons with details
func (p Undefined) Verbose() []string {
	return p.verbose
}

func copyToVerboseIfNil(short []string, verbose *[]string) {
	if verbose == nil || len(*verbose) == 0 {
		*verbose = short
	}
}
