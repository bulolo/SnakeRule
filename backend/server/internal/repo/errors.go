package repo

import "errors"

var ErrNotFound = errors.New("resource not found")
var ErrProtected = errors.New("resource is protected")
