# What the hell is going on?

Watch kubernetes resources and print the delta in changes.

## Install

`go install github.com/ibuildthecloud/wtfk8s`

## Example

```
# Watch all resources and print diffs
wtfk8s

# Watch specific resources
wtfk8s pod clusters.cluster.x-k8s.io
```
