# Simple jobqueue using Operator SDK

Currently just a wrapper around Job.

Will be expanded to ensure that Jobs can pick work items off
the workqueue and process work items to completion.

# Why do we need this?

As a pod can be terninated for a variety of reasons including

- Priority
- Node Failure
- Resource Based termination
- Program error resulting in container termination

The onus falls on the actual workload to co-ordinate/detect
when a work item is dequeued but not completed.

In a very large cluster when using Jobs that require a large
number of completions, where time taken to process a work item
is indeterminate; it adds significant complexity to the workload.

Given that Kubernetes already tracks pod failures, try and 
leverage that knowledge to make the process simpler.

The goal is for Kubernetes to work in conjunction with the
job to ensure work items are handled as efficently as possible.

The controller will detect all failed pods and hand off workitem
assigned to the failed pod to newly created pods.

The controller does not care about the actual work. Just the workid
that is handed off to the pods. The workid can then be used by the
Job to pick an item off the workqueue.
