package responsewriter

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type Request struct {
	*http.Request

	Params httprouter.Params
}

func NewRequest(r *http.Request, p httprouter.Params) *Request {
	return &Request{
		Request: r,
		Params:  p,
	}
}
