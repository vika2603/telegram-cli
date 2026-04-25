package peer

// MapResolveErrForTest re-exports the internal mapResolveErr for the
// black-box test package. This file is compiled only under `go test`,
// so the helper never leaks into production builds.
var MapResolveErrForTest = mapResolveErr

// NormalizeInputPeerIDForTest re-exports normalizeInputPeerID for
// black-box tests that verify ID normalization.
var NormalizeInputPeerIDForTest = normalizeInputPeerID
