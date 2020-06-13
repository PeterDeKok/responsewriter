package responsewriter

import (
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
	"net/http"
	"peterdekok.nl/gotools/logger"
	"peterdekok.nl/gotools/responsewriter/responsetype"
)

type Handler func(r *Request) interface{}

type ResponseType responsetype.ResponseType

var (
	log logger.Logger
)

func init() {
	log = logger.New("responsewriter")
}

func ResponseHandler(handler Handler, preferredType ResponseType, allowedTypes ...ResponseType) httprouter.Handle {
	if preferredType == nil {
		panic("Invalid response type given for response handler")
	}

	preferredAcceptedType := preferredType.GetAcceptedType()

	if len(preferredAcceptedType) == 0 {
		panic("Invalid accepted response type given")
	}

	types := make(map[string]ResponseType)

	for _, allowedType := range allowedTypes {
		types[allowedType.GetAcceptedType()] = allowedType
	}

	types[preferredAcceptedType] = preferredType

	return func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		at := req.Header.Get("Accept")

		t := preferredType

		if st, ok := types[at]; ok && len(at) > 0 && st != nil {
			t = st
		}

		resp := handler(NewRequest(req, p))

		cResp := t.Unmarshal(resp)

		if cResp == nil {
			var ok bool

			cResp, ok = resp.(responsetype.Response)

			if !ok {
				cResp = t.DefaultError()
			}
		}

		c, b := cResp.Handle()

		if b != nil {
			ct := cResp.GetContentType()

			if len(ct) == 0 {
				ct = "text/plain"
			}

			w.Header().Set("Content-Type", ct)
		} else if c == http.StatusOK {
			c = http.StatusNoContent
		}

		w.WriteHeader(c)

		if b == nil {
			return
		}

		if _, err := w.Write(b); err != nil {
			log.WithFields(logrus.Fields{
				"code":   c,
				"status": responsetype.CodeToStatus(c),
			}).WithError(err).Error("Failed to write response body")
		}
	}
}
