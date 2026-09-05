---
name: Bug report
about: Report something that behaves differently from its documentation
title: ""
labels: bug
assignees: ""
---

**What happened**

A clear and concise description of the wrong behavior.

**What you expected**

The behavior the docs promise. Link the exact doc section or godoc comment
that sets the expectation.

**Minimal reproduction**

```go
// smallest program that shows the bug
```

**Version**

- go-idempotency: (output of `go list -m github.com/larsartmann/go-idempotency`)
- Go: (output of `go version`)
- Backend you implement `Store` against, if relevant

**Evidence**

Run `go test ./... -race -count=1` in your reproduction and paste the failure.
If the contract suite (`contract.RunTests`) fails against your backend, say
which subtest.
