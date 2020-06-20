package responsetype

import (
	"github.com/sirupsen/logrus"
	"net/http"
)

type Response interface {
	Handle() (int, []byte)
	GetBody() []byte
	GetContentType() string
	Coder
}

type Coder interface {
	GetCode() int
}

type TypeHandler interface {
	GetAcceptedType() string
	Unmarshal(resp interface{}) Response
	DefaultError() Response
	String() string
}

type ResponseType TypeHandler

var (
	TypeJSON ResponseType = &JSON{}
	//TypePlainText =
)

func CodeToLogLevel(code int) logrus.Level {
	switch {
	case code < 200: // Invalid
		return logrus.TraceLevel
	case code < 300: // 2xx
		return logrus.DebugLevel
	case code < 400: // 3xx
		return logrus.InfoLevel
	case code < 500: // 4xx
		return logrus.WarnLevel
	case code < 600: // 5xx
		return logrus.ErrorLevel
	default: // Invalid
		return logrus.TraceLevel
	}
}

func CodeToStatus(code int) string {
	sts := http.StatusText(code)

	if len(sts) == 0 {
		sts = "Unknown error"
	}

	return sts
}
