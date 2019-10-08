package client

import (
  "reflect"
  "github.com/mitchellh/mapstructure"
)

type ErrorResponse struct {
  Code  int    `json:"code" binding:"required"`
  Error string `json:"error" binding:"required"`
}

type Response struct {
  Index  int             `json:"index" validate:"min=0"`
  Status int             `json:"status" validate:"required"`
  Errors []ErrorResponse `json:"errors" validate:"dive"`
  Ok     interface{}     `json:"ok" validate:"dive"`
}

type Responses []Response

func (r Responses) Unmarshal(i int, v interface{}) (rStatus int, rErr []ErrorResponse) {
  conf := &mapstructure.DecoderConfig{
      WeaklyTypedInput: false,
      Result:           &v,
      TagName:          "json",
  }

  decoder, err := mapstructure.NewDecoder(conf)
  if err != nil {
      panic(err)
  }

  responses := reflect.ValueOf(r)

  for x := 0; x < responses.Len(); x++ {
    response := responses.Index(x)

    index := response.FieldByName("Index").Interface().(int)
    if index == i {
      // found response with given index

      status  := response.FieldByName("Status")
      e       := response.FieldByName("Errors")
      ok      := response.FieldByName("Ok")

      if status.CanInterface() {
        rStatus = status.Interface().(int)
      }

      if ok.CanInterface() {
        iOk := ok.Interface()
        err = decoder.Decode(iOk)
        if err != nil {
            panic(err)
        }
      }

      if e.CanInterface() {
        rErr = e.Interface().([]ErrorResponse)
      }

      return rStatus, rErr
    }

  }

  panic("Given index not found")
}
