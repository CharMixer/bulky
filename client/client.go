package client

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
