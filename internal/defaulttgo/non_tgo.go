//go:build !defaulttgo

// This package is a helper package to simplify testing of the opt-in tgo support in
// the scanner and parser packages. It helps us run the non-tgo test suite against
// the tgo-enabled scanner and parser, without the need to rewrite all the upstream tests.
//
// The default-tgo mode is enabled by building the binary with the defaulttgo build tag and by calling
// [Enable]. We require such additional call such that external users of our APIs cannot turn this mode on.
//
// In default-tgo mode the Parser/Scanner are by default working in the "tgo-mode" (ParseTgo/ScanTgo bit is set implicitly).
package defaulttgo

const Enabled = false

func Enable() {}
