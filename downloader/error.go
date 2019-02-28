package downloader

import (
	"errors"
	"fmt"
	"strconv"
)

var (
	ErrInvalidItemID = errors.New("downloader: invalid item id")
	ErrNotFound      = errors.New("downloader: not found")
	ErrUnsupported   = errors.New("downloader: unsupported type")
	ErrOutOfRange    = errors.New("downloader: out of range")
)

type HTTPError struct {
	Code int
}

func (e *HTTPError) Error() string {
	return "http status code " + strconv.Itoa(e.Code)
}

type ContentLengthError struct {
	Expected, Got int64
}

func (e *ContentLengthError) Error() string {
	return fmt.Sprintf("expected length %d, got %d", e.Expected, e.Got)
}
