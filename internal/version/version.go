package version

// GitSHA identifies the build this binary came from. Overridden at build
// time via -ldflags "-X .../internal/version.GitSHA=<sha>" (see the
// Makefile's build-backend target and the Dockerfile); left as "dev" for a
// plain `go build`/`go run` with no ldflags, e.g. local development.
var GitSHA = "dev"
