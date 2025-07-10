---
title: Usage
expires_at: never
tags: [diego-release, debugserver]
---

## Usage  

To test changes to debugserver code in Concourse pipeline, perform the following.
A git commit to this repo runs a pipeline test and it is expected to pass. 
Running it locally before pushing your commit will ensure the pipeline run will succeed.

```
$ cd <path-to-debugserver-repo>
$ fly login -c <> -t <> -n <>
$ fly -t <> execute --input debugserver=./ --config ./tmp/test-debugserver.yml | tee /tmp/debugserver-fly-exec.log
```

Create the task file in any directory. For example:
```
$ cat /tmp/test-debugserver.yml
---
platform: linux

image_resource:
  type: registry-image
  source: {repository: golang, tag: "1.23"}

inputs:
  - name: debugserver

run:
  path: bash
  args:
    - -c
    - |
      cd debugserver
      go test -v ./...
```

References:
https://concourse-ci.org/tasks.html#running-tasks

To test changes to debugserver locally without using the `fly`, perform the following.

```
$ cd <path-to-debugserver-repo>
$ go test -v ./...
```
