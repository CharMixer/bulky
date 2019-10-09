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
        request.Output = NewClientErrorResponse(request.Index, "MAX_REQUESTS_EXCEEDED")
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
          request.Output = NewClientErrorResponse(request.Index, "EMPTY_REQUEST_NOT_ALLOWED")

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
            errorResponses = append(errorResponses, client.ErrorResponse{Code: E.MAP["INPUT_VALIDATION_FAILED"].Code, Error: e.Translate(nil)})
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
          request.Output = NewClientErrorResponse(request.Index, "FAILED_DUE_TO_OTHER_ERRORS")
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
    request.Output = NewInternalErrorResponse(request.Index, "OPERATION_ABORTED")
  }

  return errors.New("Output validation failed")
}
func NewInternalErrorResponse(index int, code... string) (*client.Response) {

  if code == nil {
    code = append(code, "INTERNAL_SERVER_ERROR")
  }

  var data []client.ErrorResponse
  for _, c := range code {
    e := E.MAP[c]
    data = append(data, client.ErrorResponse{Code: e.Code, Error: e.Error["en"]})
  }

  return &client.Response{
    Index: index,
    Status: http.StatusInternalServerError,
    Errors: data,
  }
}
func NewClientErrorResponse(index int, code... string) (*client.Response) {
  if code == nil {
    panic("No errors defined for client error response")
  }

  var data []client.ErrorResponse
  for _, c := range code {
    e := E.MAP[c]
    data = append(data, client.ErrorResponse{Code: e.Code, Error: e.Error["en"]})
  }

  return &client.Response{
    Index: index,
    Status: http.StatusNotFound,
    Errors: data,
  }
}
func NewOkResponse(index int, data interface{}) (*client.Response) {
  return &client.Response{
    Index: index,
    Status: http.StatusNotFound,
    Errors: []client.ErrorResponse{},
    Ok: data,
  }
}
func FailAllRequestsWithClientErrorResponse(requests []*Request, code... string) {
  for _,r := range requests {
    r.Output = NewClientErrorResponse(r.Index, code...)
  }
}
func FailAllRequestsWithInternalErrorResponse(requests []*Request, code... string) {
  for _,r := range requests {
    r.Output = NewInternalErrorResponse(r.Index, code...)
  }
}
