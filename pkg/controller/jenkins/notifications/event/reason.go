package event

import "fmt"

const (
	// OperatorSource defines that notification concerns operator
	OperatorSource Source = "operator"
	// KubernetesSource defines that notification concerns kubernetes
	KubernetesSource Source = "kubernetes"
)

// Reason is interface that let us know why operator sent notification
type Reason interface {
	Short() []string
	Verbose() []string
}

// PodRestartReason defines the reason why Jenkins master pod restarted
type PodRestartReason struct {
	source  Source
	short   []string
	verbose []string
}

// NewPodRestartReason returns new instance of PodRestartReason
func NewPodRestartReason(source Source, short []string, verbose []string) *PodRestartReason {
	return &PodRestartReason{source: source, short: short, verbose: verbose}
}

// Source is enum type that informs us what triggered notification
type Source string

// Short is list of reasons
func (p PodRestartReason) Short() []string {
	return append([]string{fmt.Sprintf("pod restarted by: '%s'", p.source)}, p.short...)
}

// Verbose is list of reasons with details
func (p PodRestartReason) Verbose() []string {
	return append([]string{fmt.Sprintf("pod restarted by: '%s'", p.source)}, p.verbose...)
}
