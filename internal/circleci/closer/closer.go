// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

/*
Package closer contains a helper function for not losing deferred errors
*/
package closer

import "io"

// ErrorHandler closes an io.Closer with error handling. If there's an
// error during Close when `in` is `nil`, sets `in` to the value of
// the Close error.
func ErrorHandler(c io.Closer, in *error) {
	cerr := c.Close()
	if *in == nil {
		*in = cerr
	}
}
