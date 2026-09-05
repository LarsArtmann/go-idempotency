---
name: Feature request
about: Propose an addition to the SDK
title: ""
labels: enhancement
assignees: ""
---

**Problem first**

What can you not do today? Describe the domain need, not the implementation.
Who else hits this — is it specific to one deployment shape?

**Why it belongs in this module**

This library deliberately owns only the `Store` contract, error semantics, and
the contract test suite (see ADR-001: no production backends, ever). Explain
why the request is contract/surface work rather than something your code can
compose on top of the existing interface. If a doc-level recipe would solve
it, say so — those are usually preferred.

**Proposed shape**

Sketch the API as you'd want to call it (types, signatures, godoc). Note
SemVer impact: new methods on `Store` are additive but force every backend
implementer to change; prefer compositions that do not.

**Willing to contribute?**

Say if you plan to open a PR. Include tests with the change — contract-suite
coverage for anything touching `Store` semantics.
