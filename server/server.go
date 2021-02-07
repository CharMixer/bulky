package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"time"

	"github.com/charmixer/bulky/client"
	E "github.com/charmixer/bulky/errors"
	"github.com/go-playground/validator/v10"
)

type Request struct {
	Index  int
	Input  interface{}
	Output *client.Response
}
type HandleRequestParams struct {
	EnableEmptyRequest      bool
	DisableInputValidation  bool
	DisableOutputValidation bool
	MaxRequests             int64
	Debug                   bool
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
			Index:  index,
			Input:  request,
			Output: nil, // someone needs to fill this
		})
	}

	start = time.Now()

	if params.MaxRequests != 0 && int64(len(requests)) > params.MaxRequests { // deny all requests if too many was given
		// fail all
		for _, request := range requests {
			if request.Output == nil {
				request.Output = NewBadRequestErrorResponse(request.Index, E.MAX_REQUESTS_EXCEEDED)
			}

			responses = append(responses, request.Output)
		}

		return responses
	}

	if !params.DisableInputValidation {

		var errorsFound = false

		for _, request := range requests {

			if request.Input == nil {
				// if we dont allow the empty set, return an error to the user
				if !params.EnableEmptyRequest {
					request.Output = NewBadRequestErrorResponse(request.Index, E.EMPTY_REQUEST_NOT_ALLOWED)

					errorsFound = true
					continue
				}
			}

			// validate requests
			if request.Input != nil { // if not the empty set, then validate
				err := E.Validate.Struct(request.Input)
				if err != nil {

					var errorResponses []client.ErrorResponse
					for _, e := range err.(validator.ValidationErrors) {
						errorResponses = append(errorResponses, client.ErrorResponse{Code: E.INPUT_VALIDATION_FAILED, Error: e.Translate(nil)})
					}

					request.Output = &client.Response{
						Index:  request.Index,
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
			for _, request := range requests {
				if request.Output == nil {
					request.Output = NewClientOperationAbortedResponse(request.Index)
				}

				responses = append(responses, request.Output)
			}

			return responses
		}

	}

	inputValidationTime := time.Since(start)

	// handle requests
	start = time.Now()
	iHandleRequests(requests)
	handleRequestsTime := time.Since(start)

	var outputValidationTime time.Duration
	if !params.DisableOutputValidation {
		start = time.Now()
		_ = OutputValidateRequests(requests)
		outputValidationTime = time.Since(start)
	}

	for _, request := range requests {
		if request.Output == nil {
			panic("Not all requests have been handled")
		}
		responses = append(responses, request.Output)
	}

	if params.Debug {
		fmt.Println("========== BULKY DEBUGGING BEGIN ==========")
		pc, file, no, ok := runtime.Caller(1)
		if ok {
			details := runtime.FuncForPC(pc)
			if details != nil {
				fmt.Printf("%s (%s:%d)\n", details.Name(), file, no)
			}
		}
		logRequests(requests)
		fmt.Printf("Input validation: %v, HandleRequests: %v, Output validation: %v\n", inputValidationTime, handleRequestsTime, outputValidationTime)
		fmt.Println("========== BULKY DEBUGGING END ==========")
	}

	return responses
}
func OutputValidateRequests(requests []*Request) error {
	var passedRequests []*Request

	for _, request := range requests {
		if request.Output == nil {
			panic("Not all requests have been handled")
		}

		// output validation
		err := E.Validate.Struct(request.Output)
		if err != nil {
			i, _ := json.MarshalIndent(request.Input, "", "  ")
			o, _ := json.MarshalIndent(request.Output, "", "  ")
			fmt.Printf("ATTENTION! Response validation failed. \nErrors:\n%s\n\nRequest: %s\n\nResponse: %s\n", err.Error(), i, o)

			request.Output = NewInternalErrorResponse(request.Index)
			continue
		}

		passedRequests = append(passedRequests, request)
	}

	if len(passedRequests) == len(requests) {
		return nil
	}

	// deny by default
	for _, request := range passedRequests {
		request.Output = NewServerOperationAbortedResponse(request.Index)
	}

	return errors.New("Output validation failed")
}

func logRequests(requests []*Request) {
	for _, req := range requests {
		request, err := json.MarshalIndent(req.Input, "", "  ")

		if err != nil {
			fmt.Println(err.Error())
		}

		response, err := json.MarshalIndent(req.Output, "", "  ")
		if err != nil {
			fmt.Println(err.Error())
		}

		fmt.Printf("[index: %d] => %s -> %s\n\n", req.Index, request, response)
	}
}

func NewErrorResponse(index int, status int, code ...int) *client.Response {
	var data []client.ErrorResponse
	for _, c := range code {
		e := E.MAP[c]
		data = append(data, client.ErrorResponse{Code: c, Error: e["en"]})
	}

	return &client.Response{
		Index:  index,
		Status: status,
		Errors: data,
		Ok:     nil,
	}
}
func NewClientErrorResponse(index int, code ...int) *client.Response {
	if code == nil {
		panic("No errors defined for client error response")
	}

	return NewErrorResponse(index, http.StatusNotFound, code...)
}
func NewBadRequestErrorResponse(index int, code ...int) *client.Response {
	if code == nil {
		panic("No errors defined for client error response")
	}

	return NewErrorResponse(index, http.StatusBadRequest, code...)
}
func NewInternalErrorResponse(index int) *client.Response {
	return NewErrorResponse(index, http.StatusInternalServerError, E.INTERNAL_SERVER_ERROR)
}
func NewServiceUnavailableResponse(index int) *client.Response {
	return NewErrorResponse(index, http.StatusServiceUnavailable)
}
func NewServerOperationAbortedResponse(index int) *client.Response {
	return NewErrorResponse(index, http.StatusInternalServerError, E.OPERATION_ABORTED)
}
func NewClientOperationAbortedResponse(index int) *client.Response {
	return NewClientErrorResponse(index, http.StatusNotFound, E.OPERATION_ABORTED)
}

func FailAllRequestsWithErrorResponse(requests []*Request, status int, code ...int) {
	for _, r := range requests {
		r.Output = NewErrorResponse(r.Index, status, code...)
	}
}
func FailAllRequestsWithClientOperationAbortedResponse(requests []*Request) {
	FailAllRequestsWithErrorResponse(requests, http.StatusNotFound, E.OPERATION_ABORTED)
}
func FailAllRequestsWithServerOperationAbortedResponse(requests []*Request) {
	FailAllRequestsWithErrorResponse(requests, http.StatusInternalServerError, E.OPERATION_ABORTED)
}
func FailAllRequestsWithClientErrorResponse(requests []*Request, code ...int) {
	FailAllRequestsWithErrorResponse(requests, http.StatusNotFound, code...)
}
func FailAllRequestsWithInternalErrorResponse(requests []*Request) {
	FailAllRequestsWithErrorResponse(requests, http.StatusInternalServerError, E.INTERNAL_SERVER_ERROR)
}
func FailAllRequestsWithServiceUnavailableResponse(requests []*Request) {
	FailAllRequestsWithErrorResponse(requests, http.StatusServiceUnavailable)
}

func NewOkResponse(index int, data interface{}) *client.Response {
	return &client.Response{
		Index:  index,
		Status: http.StatusOK,
		Errors: nil,
		Ok:     data,
	}
}
