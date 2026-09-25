module github.com/iv-one/x

go 1.27

require (
	github.com/rs/xid v1.6.0
	github.com/stretchr/testify v1.12.1
	// MUST stay >= v0.43.0: webp.DecodeConfig panicked on a malformed container
	// before it (CVE-2026-46601), which is the header pre-check imagex relies on.
	// A dependency to watch: recurring panic class (GO-2026-4815).
	golang.org/x/image v0.46.0
	golang.org/x/net v0.59.0
)

require (
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/text v0.42.0 // indirect
)
