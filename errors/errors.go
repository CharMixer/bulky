package errors

const INTERNAL_SERVER_ERROR      = -1
const EMPTY_REQUEST_NOT_ALLOWED  = -2
const MAX_REQUESTS_EXCEEDED      = -3
const OPERATION_ABORTED          = -4
const INPUT_VALIDATION_FAILED    = -5


var MAP = map[int]map[string]string {
  INTERNAL_SERVER_ERROR: map[string]string{
    "en"  : "Internal server error occured. Please wait until it has been fixed, before you try again",
    "dev" : "Internal server error occured. Please wait until it has been fixed, before you try again",
  },
  EMPTY_REQUEST_NOT_ALLOWED: map[string]string{
    "en"  : "Empty request not allowed",
    "dev" : "This endpoint does not allow the empty request - each request must be defined separately",
  },
  MAX_REQUESTS_EXCEEDED: map[string]string{
    "en"  : "Max number of requests exceeded",
    "dev" : "MaxRequest parameter has been set for endpoint and is exceeded by the number of request-objects given in the input",
  },
  OPERATION_ABORTED: map[string]string{
    "en"  : "Operation aborted due to other errors",
    "dev" : "Operation aborted due to other errors",
  },
  INPUT_VALIDATION_FAILED: map[string]string{
    "en"  : "Input validation failed",
    "dev" : "Struct validations failed on tags for input",
  },
}

func AppendError(code int, iError map[string]string) {
  if _, ok := MAP[code]; ok {
    panic("Error code '" + string(code) + "' already defined")
  }

  MAP[code] = iError
}

func AppendErrors(iErrors map[int]map[string]string) {
  for i,e := range iErrors {
    AppendError(i,e)
  }
}
