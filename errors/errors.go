package errors

import (
  "errors"
)

type ErrorDefinition struct {
  Code int
  Error map[string]string
}

var MAP = map[string]ErrorDefinition{
  "INTERNAL_SERVER_ERROR": {
    Code: -1,
    Error: map[string]string{
      "en"  : "Internal server error occured. Please wait until it has been fixed, before you try again",
      "dev" : "Internal server error occured. Please wait until it has been fixed, before you try again",
    },
  },
  "EMPTY_REQUEST_NOT_ALLOWED": {
    Code: -2,
    Error: map[string]string{
      "en"  : "Empty request not allowed",
      "dev" : "This endpoint does not allow the empty request - each request must be defined separately",
    },
  },
  "MAX_REQUESTS_EXCEEDED": {
    Code: -3,
    Error: map[string]string{
      "en"  : "Max number of requests exceeded",
      "dev" : "MaxRequest parameter has been set for endpoint and is exceeded by the number of request-objects given in the input",
    },
  },
  "OPERATION_ABORTED": {
    Code: -4,
    Error: map[string]string{
      "en"  : "Operation aborted due to other errors",
      "dev" : "Operation aborted due to other errors",
    },
  },
  "INPUT_VALIDATION_FAILED": {
    Code: -5,
    Error: map[string]string{
      "en"  : "Input validation failed",
      "dev" : "Struct validations failed on tags for input",
    },
  },
}

func ValidateErrorsIntegrity() (error) {
  return errors.New("Something doesnt look right")
}

func AppendError(i string, iError ErrorDefinition) {
  if _, ok := MAP[i]; ok {
    panic("Error " + i + " already defined")
  }
  MAP[i] = iError
}

func AppendErrors(iErrors map[string]ErrorDefinition) {
  for i,e := range iErrors {
    AppendError(i,e)
  }
}
