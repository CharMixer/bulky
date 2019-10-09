package server

import (
  "fmt"
  "errors"
  "reflect"
  "time"
  "encoding/json"
  "net/http"
  "gopkg.in/go-playground/validator.v9"

  "github.com/charmixer/bulky/client"
  E "github.com/charmixer/bulky/errors"
)

var validate = validator.New()

type Request struct {
  Index int
  Input interface{}
  Output *client.Response
}
type HandleRequestParams struct {
  EnableEmptyRequest bool
  DisableInputValidation bool
  DisableOutputValidation bool
  MaxRequests int64
}
type IHandleRequests func(requests []*Request)
func HandleRequest(iRequests interface{}, iHandleRequests IHandleRequests, params HandleRequestParams) (responses []interface{}) {
  var requests []*Request

  var start time.Time

  // initialize structs
  tmpRequests := reflect.ValueOf(iRequests)
  for index := 0; index < tmpRequests.Len() || (tmpRequests.Len() == 0 && index == 0); index++ {
    var request interface{}
    if tmpRequests.Len() > 0 {
      request = tmpRequests.Index(index).Interface()
    }
    requests = append(requests, &Request{
      Index: index,
      Input: request,
      Output: nil, // someone needs to fill this
    })
  }

  start = time.Now()

  if params.MaxRequests != 0 && int64(len(requests)) > params.MaxRequests { // deny all requests if too many was given
    // fail all
    for _,request := range requests {
      if request.Output == nil {
        request.Output = NewClientErrorResponse(request.Index, E.MAX_REQUESTS_EXCEEDED)
      }

      responses = append(responses, request.Output)
    }

    return responses
  }

  if !params.DisableInputValidation {

    var errorsFound = false

    for _,request := range requests {

      if request.Input == nil {
        // if we dont allow the empty set, return an error to the user
        if !params.EnableEmptyRequest {
          request.Output = NewClientErrorResponse(request.Index, E.EMPTY_REQUEST_NOT_ALLOWED)

          errorsFound = true
          continue
        }
      }

      // validate requests
      if request.Output != nil { // if not the empty set, then validate
        err := validate.Struct(request.Output)
        if err != nil {

          var errorResponses []client.ErrorResponse
          for _, e := range err.(validator.ValidationErrors) {
            errorResponses = append(errorResponses, client.ErrorResponse{Code: E.INPUT_VALIDATION_FAILED, Error: e.Translate(nil)})
          }

          request.Output = &client.Response{
            Index: request.Index,
            Status: http.StatusBadRequest,
            Errors: errorResponses,
          }

          errorsFound = true
          continue
        }
      }

    }

    if errorsFound { // make sure if something fails, others will fail too
      // fail all
      for _,request := range requests {
        if request.Output == nil {
          request.Output = NewClientErrorResponse(request.Index, E.OPERATION_ABORTED)
        }

        responses = append(responses, request.Output)
      }

      return responses
    }

  }
  fmt.Printf("%s took %v\n", "input validation", time.Since(start))

  // handle requests
  start = time.Now()
  iHandleRequests(requests)
  fmt.Printf("%s took %v\n", "iHandleRequests", time.Since(start))

  if !params.DisableOutputValidation {
    start = time.Now()
    _ = OutputValidateRequests(requests)
    fmt.Printf("%s took %v\n", "output validation took", time.Since(start))
  }

  for _,request := range requests {
    if request.Output == nil {
      panic("Not all requests have been handled")
    }
    responses = append(responses, request.Output)
  }

  return responses
}
func OutputValidateRequests(requests []*Request) (error){
  var passedRequests []*Request

  for _,request := range requests {
    if request.Output == nil {
      panic("Not all requests have been handled")
    }

    // output validation
    err := validate.Struct(request.Output)
    if err != nil {
      i, _ := json.MarshalIndent(request.Input, "", "  ")
      o, _ := json.MarshalIndent(request.Output, "", "  ")
      fmt.Printf("ATTENTION! Response validation failed. \nErrors:\n%s\n\nRequest: %s\n\nResponse: %s\n", err.Error(), i, o)

      request.Output = NewInternalErrorResponse(request.Index)
      continue;
    }

    passedRequests = append(passedRequests, request)
  }

  if len(passedRequests) == len(requests) {
    return nil
  }

  // deny by default
  for _,request := range passedRequests {
    request.Output = NewInternalErrorResponse(request.Index, E.OPERATION_ABORTED)
  }

  return errors.New("Output validation failed")
}
func NewErrorResponse(index int, status int, code... int) (*client.Response) {
  var data []client.ErrorResponse
  for _, c := range code {
    e := E.MAP[c]
    data = append(data, client.ErrorResponse{Code: c, Error: e["en"]})
  }

  return &client.Response{
    Index: index,
    Status: status,
    Errors: data,
  }
}
func NewInternalErrorResponse(index int, code... int) (*client.Response) {
  if code == nil {
    // should we force this on developer, eg panic
    code = append(code, E.INTERNAL_SERVER_ERROR)
  }

  return NewErrorResponse(index, http.StatusInternalServerError, code...)
}
func NewClientErrorResponse(index int, code... int) (*client.Response) {
  if code == nil {
    panic("No errors defined for client error response")
  }

  return NewErrorResponse(index, http.StatusNotFound, code...)
}
func NewOkResponse(index int, data interface{}) (*client.Response) {
  return &client.Response{
    Index: index,
    Status: http.StatusNotFound,
    Errors: []client.ErrorResponse{},
    Ok: data,
  }
}
func FailAllRequestsWithErrorResponse(requests []*Request, status int, code... int) {
  for _,r := range requests {
    r.Output = NewErrorResponse(r.Index, status, code...)
  }
}
func FailAllRequestsWithOperationAbortedResponse(requests []*Request) {
  FailAllRequestsWithErrorResponse(requests, http.StatusNotFound, E.OPERATION_ABORTED)
}
func FailAllRequestsWithClientErrorResponse(requests []*Request, code... int) {
  FailAllRequestsWithErrorResponse(requests, http.StatusNotFound, code...)
}
func FailAllRequestsWithInternalErrorResponse(requests []*Request) {
  FailAllRequestsWithErrorResponse(requests, http.StatusInternalServerError, E.INTERNAL_SERVER_ERROR)
}
func FailAllRequestsWithServiceUnavailableResponse(requests []*Request) {
  FailAllRequestsWithErrorResponse(requests, http.StatusServiceUnavailable)
}
