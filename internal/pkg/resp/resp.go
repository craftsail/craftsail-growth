// SPDX-License-Identifier: AGPL-3.0-or-later

package resp

import "net/http"

// Envelope is the JSON body for every API response.
type Envelope struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func OK(data any) Envelope {
	if data == nil {
		data = map[string]any{}
	}
	return Envelope{Code: 0, Msg: "ok", Data: data}
}

func Fail(code, httpStatus int, msg string) (Envelope, int) {
	if httpStatus == 0 {
		httpStatus = http.StatusBadRequest
	}
	return Envelope{Code: code, Msg: msg, Data: nil}, httpStatus
}

const (
	CodeBadRequest   = 4000
	CodeInvalidSlug  = 4001
	CodeConflict     = 4009
	CodeUnauthorized = 4010
	CodeForbidden    = 4030
	// CodePasswordChange: the default password must be replaced first.
	CodePasswordChange = 4031
	CodeNotFound       = 4040
	CodeLocked         = 4090
	CodeInternal       = 5000
)
