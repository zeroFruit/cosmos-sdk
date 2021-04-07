package errors

import "github.com/coinbase/rosetta-sdk-go/types"

var (
	NotFound = &types.Error{
		Code:        404,
		Message:     "not found",
		Description: nil,
		Retriable:   false,
		Details:     nil,
	}
)
