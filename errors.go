// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package githubapp

var (
	_ error = Error("")
)

// Error is immutable error representation.
//
// Error strings themselves are not part of semver compatibility guarantees.
// Use exported symbols instead of error strings.
type Error string

// Implements Error() interface.
func (e Error) Error() string {
	return string(e)
}
