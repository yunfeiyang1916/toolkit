package rpcerror

import (
	"encoding/json"
)

// HTTPError define rpc error struct from HTTP transport.
type HTTPError struct {
	C    int    `json:"code,omitempty"`
	Desc string `json:"error,omitempty"`
}

func HTTP(body []byte) RPCError {
	he := HTTPError{}
	json.Unmarshal(body, &he)
	return New(he.C, he.Desc)
}

func (r HTTPError) Marshal() string {
	b, _ := json.Marshal(r)
	return string(b)
}
