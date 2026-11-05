package cluster

import "errors"

func errorsAs[T error](err error, target *T) bool { return errors.As(err, target) }
