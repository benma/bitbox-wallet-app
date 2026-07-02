// Copyright (C) 2026 BitBoxSwiss.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build sqlcipher && !libsqlcipher && !darwin
// +build sqlcipher,!libsqlcipher,!darwin

package libtomcrypt

/*
#cgo CFLAGS: -DLTC_SOURCE
#cgo CFLAGS: -I${SRCDIR}
#cgo windows LDFLAGS: -ladvapi32
*/
import "C"
