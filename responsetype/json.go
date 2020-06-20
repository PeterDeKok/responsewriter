package responsetype

import (
	"bytes"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"net/http"
	"peterdekok.nl/gotools/logger"
	"reflect"
)

type JSON struct {
	Code int
	Body interface{}
	err  error
	log  logger.Logger
}

type JSONResponsable interface {
	ToJSON() *JSON
}

var (
	log                          logger.Logger
	InternalServerErrorJsonBytes []byte
)

func init() {
	log = logger.New("responsewriter.responsetype.json")

	InternalServerErrorJsonBytes, _ = json.Marshal((&JSON{}).defaultError())
}

func NewJSON(code int, body interface{}) *JSON {
	return &JSON{
		Code: code,
		Body: body,
	}
}

func (r *JSON) WithError(err error) *JSON {
	r.err = err

	return r
}

func (r *JSON) WithLogger(log logger.Logger) *JSON {
	r.log = log

	return r
}

func (r JSON) GetCode() int {
	return r.Code
}

func (r JSON) GetBody() []byte {
	if r.Body == nil {
		return []byte{}
	}

	switch b := r.Body.(type) {
	case []byte:
		return b
	case string:
		return []byte(b)
	}

	if b, err := json.Marshal(r); err == nil {
		return b
	}

	log.Warn("Failed to marshal json response")

	return InternalServerErrorJsonBytes
}

func (r JSON) Handle() (int, []byte) {
	c := r.GetCode()
	b := r.GetBody()

	if c == 0 && len(b) == 0 {
		c = http.StatusInternalServerError
		b = InternalServerErrorJsonBytes
	} else if bytes.Compare(b, InternalServerErrorJsonBytes) == 0 {
		c = http.StatusInternalServerError
	} else if c == 0 {
		c = http.StatusOK
	}

	if r.err != nil && r.log != nil {
		r.log.WithFields(logrus.Fields{
			"code":   c,
			"status": CodeToStatus(c),
		}).WithError(r.err).Log(CodeToLogLevel(c), "Response error encountered")
	}

	return c, b
}

func (r JSON) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Body)
}

func (r JSON) GetContentType() string {
	return "application/json"
}

func (r *JSON) Unmarshal(resp interface{}) Response {
	if resp == nil {
		return &JSON{Code: http.StatusNoContent, Body: []byte{}}
	}

	if jr, ok := resp.(JSONResponsable); ok {
		return jr.ToJSON()
	}

	switch cResp := resp.(type) {
	case string:
		if len(cResp) == 0 {
			return &JSON{Code: http.StatusNoContent, Body: []byte{}}
		}

		return &JSON{Code: http.StatusOK, Body: cResp}
	case JSON:
		return &cResp
	case *JSON:
		return cResp
	case JSONError:
		return &JSON{Code: cResp.Code, Body: cResp, err:  cResp.Err}
	case *JSONError:
		return &JSON{Code: cResp.Code, Body: cResp, err:  cResp.Err}
	case int:
		return &JSON{Code: cResp}
	case int32:
		return &JSON{Code: int(cResp)}
	case int16:
		return &JSON{Code: int(cResp)}
	case int8:
		return &JSON{Code: int(cResp)}
	case uint16:
		return &JSON{Code: int(cResp)}
	case uint8:
		return &JSON{Code: int(cResp)}
	}

	// We assume an error adhering to the json.Marshaler interface
	// will be consumable for public r
	if err, ok := resp.(error); ok {
		j := r.defaultError()

		j.err = err

		mar, marOk := resp.(json.Marshaler)
		cod, codOk := resp.(Coder)

		if codOk {
			j.Code = cod.GetCode()
		}

		if marOk {
			j.Body = mar
		} else if codOk {
			je := r.defaultJsonError()
			je.Code = j.Code
			je.Err = err

			j.Body = je
		}

		return j
	} else if jm, ok := resp.(json.Marshaler); ok {
		c := http.StatusOK

		if coder, ok := resp.(Coder); ok {
			c = coder.GetCode()
		}

		if b, err := json.Marshal(jm); err == nil {
			return &JSON{Code: c, Body: b}
		}
	}

	switch reflect.TypeOf(resp).Kind() {
	case reflect.Map, reflect.Slice, reflect.Array:
		return &JSON{Code: http.StatusOK, Body: resp}
	}

	return nil
}

func (r *JSON) DefaultError() Response {
	return r.defaultError()
}

func (r *JSON) defaultError() *JSON {
	return &JSON{
		Code: http.StatusInternalServerError,
		Body: r.defaultJsonError(),
	}
}

func (r *JSON) defaultJsonError() JSONError {
	return JSONError{
		Code:        http.StatusInternalServerError,
		Description: http.StatusText(http.StatusInternalServerError),
	}
}

func (r *JSON) GetAcceptedType() string {
	return "application/json"
}

func (r *JSON) String() string {
	return r.GetAcceptedType()
}

// Json error wrapper
type JSONError struct {
	Code        int         `json:"code"`
	Description interface{} `json:"description"`
	Err         error       `json:"-"`
}

func NewJSONError(code int, description interface{}, err error) *JSON {
	if description == nil {
		stsTxt := http.StatusText(code)

		if len(stsTxt) > 0 {
			description = stsTxt
		}
	}

	return NewJSON(code, JSONError{
		Code:        code,
		Description: description,
		Err:         err,
	}).WithError(err)
}

func (je JSONError) GetCode() int {
	return je.Code
}
