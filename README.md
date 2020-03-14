# kjob

[![e2e](https://github.com/stefanprodan/kjob/workflows/ci/badge.svg)](https://github.com/stefanprodan/kjob/actions)
[![release](https://github.com/stefanprodan/kjob/workflows/release/badge.svg)](https://github.com/stefanprodan/kjob/actions)

**kjob** is a small utility written in Go that:
* creates a Kubernetes Job from a CronJob template
* overrides the job command if specified
* waits for job completion
* prints the pods logs
* removes the pods and the job object
* exits with status 1 if the job failed

## Usage

Download kjob binary from GitHub [releases](https://github.com/stefanprodan/kjob/releases/latest).

Create a suspended CronJob that will serve as a template:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: curl
spec:
  schedule: "*/1 * * * *"
  successfulJobsHistoryLimit: 0
  failedJobsHistoryLimit: 0
  suspend: true
  jobTemplate:
    spec:
      backoffLimit: 0
      activeDeadlineSeconds: 100
      template:
        spec:
          restartPolicy: Never
          containers:
            - name: curl
              image: curlimages/curl:7.69.0
              command:
                - sh
                - -c
                - "curl -sL flagger.app/index.yaml | grep generated"
EOF
```

Run the job with:

```text
$ kjob run -t curl -n default

generated: "2020-03-04T18:53:07.586083089Z"
```

Override the job command with:

```text
$ kjob run -t curl -c "echo 'something went wrong' && grep tag"

something went wrong
error: Job has reached the specified backoff limit
exit status 1
```

List of available flags:

```text
$ kjob run --help

Usage:
  kjob run -t cron-job-template -n namespace [flags]

Examples:
  run --template curl --command "curl -sL flagger.app/index.yaml" --cleanup=false --timeout=2m

Flags:
      --cleanup             delete job and pods after completion (default true)
  -c, --command string      override job command
  -h, --help                help for run
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    namespace of the cron job template (default "default")
      --shell string        command shell (default "sh")
  -t, --template string     cron job template name
      --timeout duration    timeout for Kubernetes operations (default 1m0s)
```
