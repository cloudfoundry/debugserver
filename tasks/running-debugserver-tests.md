To test changes to debugserver code in Concourse pipeline, perform the following.

$ cd <path-to-debugserver-repo>
$ fly login -c <> -t <> -n <>
$ fly -t <> execute --input debugserver=./ --config ./tasks/test-debugserver.yml | tee /tmp/debugserver-fly-exec.log

References:
https://concourse-ci.org/tasks.html#running-tasks

To test changes to debugserver locally, perform the following.

$ cd <path-to-debugserver-repo>
$ go test -v ./...
