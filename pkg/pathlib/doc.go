// Package pathlib offers representations of filesystem paths with semantics
// appropriate for different operating systems. Path classes are divided between
// pure paths, which provide purely computational operations without I/O, and
// concrete paths, which inherit from pure paths but also provide I/O
// operations.
//
// The package is, for the most part, a direct reimplementation of Python's
// excellent pathlib library.
package pathlib
