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
| `entity` | the `Entity` / `ID` interfaces and the storage `Scheme` that `protoc-gen-go-entity` generates; `options.proto` annotates messages with their storage options |
| `entity/ids` | the key types an entity's `id` field takes (`ID`, `UUID`, `TenantID`, `SID`, `EmailID`, `TokenID`, `Rel`, `NilID`), proto messages that implement `entity.ID` |
| `errorsx`, `errorsx/raise` | HTTP-shaped errors; errors wrapped with their call site |
| `imagefetch`, `imagex` | fetching and canonicalizing images |
| `tenancy` | the tenant carried in a context, and per-tenant caches |
| `validate` | password, email, image URL and host checks |

## Protos

`entity/options.proto` and `entity/ids/ids.proto` are published to the Buf Schema Registry as
[`buf.build/iv-one/entity`](https://buf.build/iv-one/entity), labeled with each release tag.
A buf workspace takes it with `deps: [buf.build/iv-one/entity]` in `buf.yaml`, then
`buf dep update`; with protoc, put this module's directory on the include path.

`buf generate` regenerates the Go code. CI (`.github/workflows/ci.yml`) tests, lints,
checks the generated code and `buf breaking` against the previous tag. Pushing a `vX.Y.Z`
tag releases: the Go module is the tag, and CI pushes the protos to the BSR labeled
`vX.Y.Z` with the `BUF_TOKEN` repository secret.

## License

MIT, see [LICENSE](LICENSE).
