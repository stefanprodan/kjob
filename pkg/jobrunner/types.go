package jobrunner

import "time"

type Job struct {
	// TemplateRef references a Kubernetes object that contains the job template spec.
	TemplateRef JobTemplateRef `json:"templateRef"`

	// BackoffLimit specifies the number of retries before marking this job failed.
	BackoffLimit int32 `json:"backoffLimit"`

	// Timeout specifies the duration relative to the startTime that the job may be active
	// before the system tries to terminate it.
	// +optional
	Timeout time.Duration `json:"timeout,omitempty"`

	// Command specifies the job container command wrapped in a shell.
	// +optional
	Command string `json:"command,omitempty"`

	// CommandShell specifies the linux shell that executes the command; defaults to sh.
	// +optional
	CommandShell string `json:"commandShell,omitempty"`
}

// JobTemplateRef holds the reference to a Kubernetes object.
type JobTemplateRef struct {
	// Name of the Kubernetes object.
	Name string `json:"name"`

	// Namespace of the Kubernetes object.
	Namespace string `json:"namespace"`
}

// JobResult describes the result of a Kubernetes job execution.
type JobResult struct {
	// Name of the Kubernetes job.
	Name string `json:"name"`

	// Namespace of the Kubernetes job.
	Namespace string `json:"namespace"`

	// Status describes the completion state of the job.
	// +optional
	Status *JobStatus `json:"status,omitempty"`

	// Output holds the Kubernetes pod logs collected after job completion.
	// +optional
	Output string `json:"output,omitempty"`
}

// JobStatus describes the completion state of a Kubernetes job.
type JobStatus struct {
	// Failed means the job has failed its execution.
	Failed bool `json:"failed"`

	// Message is a human readable message indicating details about the job execution result.
	Message string `json:"message"`
}
