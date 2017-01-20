package api

import (
	"fmt"

	"github.com/pborman/uuid"
)

type ErrorCode string

const (
	NO_SUCH_NODE     ErrorCode = "no_such_node"
	NODE_EXISTS      ErrorCode = "node_exists"
	INVALID_PARAM    ErrorCode = "invalid_param"
	INTERNAL         ErrorCode = "internal"
	IS_DIRECTORY     ErrorCode = "node_is_dir"
	INCONGRUOUS_HASH ErrorCode = "incongruous_hash"
	NO_SUCH_BLOCK    ErrorCode = "no_such_block"
)

type ApiResponse struct {
	Meta  ApiResponseMetadata `json:"meta,omitempty"`
	Data  interface{}         `json:"data,omitempty"`
	Error *ApiError           `json:"error,omitempty"`
}

type ApiError struct {
	Code    ErrorCode `json:"code"`
	Details string    `json:"details"`
}

func (apiError ApiError) Error() string {
	return fmt.Sprint(apiError.Code, " => ", apiError.Details)
}

type ApiResponseMetadata struct {
	RequestId string `json:"request_id"`
}

func NewErrorResponse(err *ApiError) ApiResponse {
	return ApiResponse{
		Meta:  ApiResponseMetadata{uuid.New()},
		Error: err,
	}
}

func NewDataResponse(data interface{}) ApiResponse {
	return ApiResponse{
		Meta: ApiResponseMetadata{uuid.New()},
		Data: data,
	}
}
