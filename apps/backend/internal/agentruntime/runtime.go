// Package agentruntime defines the execution-environment taxonomy
// for kandev agents — the runtime backend that hosts an agent
// subprocess and its observable properties. It is a leaf package
// so any layer (task models, registry, lifecycle, individual agent
// implementations) can speak the same vocabulary without cycles.
package agentruntime

// Runtime identifies the execution backend that hosts an agent
// subprocess. Values match historical executor.Name strings so
// existing on-disk records (ExecutorRunning.RuntimeName) remain
// compatible.
type Runtime string

const (
	RuntimeStandalone Runtime = "standalone"
)

// IsContainerized reports whether the runtime hosts the agent
// subprocess inside a filesystem-isolated container/sandbox.
// All supported runtimes run on the host.
func (r Runtime) IsContainerized() bool {
	return false
}

func (r Runtime) String() string { return string(r) }
