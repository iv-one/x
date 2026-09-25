# x

Small Go packages: helpers, entity interfaces, per-tenant context and caches,
image checks.

```sh
go get github.com/iv-one/x
```

| Package | What it holds |
| --- | --- |
| `x` | conversions, durations, domains, slices, ids, redacted strings |
| `cache` | generic in-memory caches and a registry that invalidates them together |
| `clock` | a process clock that can run ahead, for testing expiry |
| `cryptox` | random keys and passwords |
| `entity` | the `Entity` / `ID` interfaces that `protoc-gen-go-entity` output implements |
| `errorsx`, `errorsx/raise` | HTTP-shaped errors; errors wrapped with their call site |
| `imagefetch`, `imagex` | fetching and canonicalizing images |
| `tenancy` | the tenant carried in a context, and per-tenant caches |
| `validate` | password, email, image URL and host checks |
