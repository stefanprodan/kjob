# job-runner

Job runner is a small utility written in Go that:
* creates a Kubernetes Job from a CronJob template
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

Run the job:

```text
go run ./cmd/kjob/ run --kubeconfig=$HOME/.kube/config -t curl -n test
```