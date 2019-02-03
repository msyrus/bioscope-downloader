package playlist

import (
	"errors"
	"strconv"
)

var (
	ErrInvalidItemID = errors.New("playlist: invalid item id")
	ErrNotFound      = errors.New("playlist: not found")
	ErrUnsupported   = errors.New("playlist: unsupported type")
)

type HTTPError struct {
	Code int
}

func (e *HTTPError) Error() string {
	return "http status code " + strconv.Itoa(e.Code)
}
