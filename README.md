# kjob

![e2e](https://github.com/stefanprodan/kjob/workflows/ci/badge.svg)
![release](https://github.com/stefanprodan/kjob/workflows/release/badge.svg)

Job runner is a small utility written in Go that:
* creates a Kubernetes Job from a CronJob template
* overrides the job command if specified
* waits for job completion
* prints the pod logs
* removes the pods and the job object
* if the job failed it exits with status 1

## Usage

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
            - name: kubectl
              image: curlimages/curl:7.69.0
              command:
                - /bin/sh
                - -c
                - "curl -sL flagger.app | grep License"
EOF
```

Download the latest [release](https://github.com/stefanprodan/kjob/releases/latest) and run the job:

```text
$ kjob run -t curl -n default
```

Override the job command:

```text
$ kjob run -t curl -c "echo 'some error message' && grep tag"

some error message
Error running job: Job has reached the specified backoff limit
exit status 1
```

List of available arguments:

```text
$ kjob run --help

Usage:
  kjob run --template cron-job-template --namespace namespace [flags]

Examples:
  run --kubeconfig $HOME/.kube/config -t curl -c "curl -sL flagger.app | grep License" --cleanup=false

Flags:
      --cleanup             delete job and pods after completion (default true)
  -c, --command string      override container command
  -h, --help                help for run
      --kubeconfig string   path to the kubeconfig file (default "/Users/aleph/.kube/config")
  -n, --namespace string    namespace of the CronJob used as template (default "default")
  -t, --template string     CronJob name used as template
```
