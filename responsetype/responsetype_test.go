package responsetype

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"peterdekok.nl/gotools/logger"
	"testing"
)

type CoderError int

func (ce CoderError) Error() string {
	return fmt.Sprintf("testerror: %d", ce)
}

func (ce CoderError) GetCode() int {
	return int(ce)
}

type JSONMarshalError string

func (jme JSONMarshalError) Error() string {
	return fmt.Sprintf("testerror: %s", string(jme))
}

func (jme JSONMarshalError) MarshalJSON() ([]byte, error) {
	return []byte("\"" + jme.Error() + "\""), nil
}

type JSONMarshalCoderError string

func (jmce JSONMarshalCoderError) Error() string {
	return fmt.Sprintf("testerror: %s", string(jmce))
}

func (jmce JSONMarshalCoderError) GetCode() int {
	return len(string(jmce))
}

func (jmce JSONMarshalCoderError) MarshalJSON() ([]byte, error) {
	return []byte("\"" + jmce.Error() + "\""), nil
}

type LogMock struct {
	*logrus.Entry
}

var logMockCalled int

func (l LogMock) WithFields(fields logrus.Fields) *logrus.Entry {
	logMockCalled++

	return l.Entry.WithFields(fields)
}

func TestCodeToLogLevel(t *testing.T) {
	expected := map[int]logrus.Level{
		0:   logrus.TraceLevel,
		101: logrus.TraceLevel,
		199: logrus.TraceLevel,
		200: logrus.DebugLevel,
		299: logrus.DebugLevel,
		300: logrus.InfoLevel,
		399: logrus.InfoLevel,
		400: logrus.WarnLevel,
		499: logrus.WarnLevel,
		500: logrus.ErrorLevel,
		599: logrus.ErrorLevel,
		600: logrus.TraceLevel,
	}

	for code, level := range expected {
		lvl := CodeToLogLevel(code)

		if lvl != level {
			t.Errorf("Wrong log level for code: %d, expected %s, got %s", code, level, lvl)
		}
	}
}

func TestCodeToStatus(t *testing.T) {
	expected := map[int]string{
		0:                              "Unknown error",
		http.StatusOK:                  http.StatusText(http.StatusOK),
		http.StatusNoContent:           http.StatusText(http.StatusNoContent),
		http.StatusFound:               http.StatusText(http.StatusFound),
		http.StatusNotAcceptable:       http.StatusText(http.StatusNotAcceptable),
		http.StatusInternalServerError: http.StatusText(http.StatusInternalServerError),
		600:                            "Unknown error",
	}

	for code, status := range expected {
		sts := CodeToStatus(code)

		if sts != status {
			t.Errorf("Wrong status for code: %d, expected %s, got %s", code, status, sts)
		}
	}
}

func TestNewJSON(t *testing.T) {
	rt := NewJSON(http.StatusFailedDependency, "testbody")

	ec := http.StatusFailedDependency
	c := rt.Code

	if c != ec {
		t.Errorf("invalid new json code, expected %d, got %d", ec, c)
	}

	eb := "testbody"
	b := rt.Body

	if b != eb {
		t.Errorf("invalid new json body, expected %s, got %s", eb, b)
	}
}

func TestJSON_WithError(t *testing.T) {
	rt := NewJSON(http.StatusFailedDependency, "testbody")

	err := errors.New("testerror")

	rtB := rt.WithError(err)

	if rt != rtB {
		t.Error("expected response to be fluent")
	}

	ee := err.Error()
	e := rt.err.Error()

	if e != ee {
		t.Errorf("invalid new json error, expected %s, got %s", ee, e)
	}
}

func TestJSON_WithLogger(t *testing.T) {
	rt := NewJSON(http.StatusFailedDependency, "testbody")

	log := logger.New("testlogger")

	rtC := rt.WithLogger(log)

	if rt != rtC {
		t.Error("expected response to be fluent")
	}

	el := log
	l := rt.log

	if l != el {
		t.Errorf("invalid new json logger, expected %s, got %s", el, l)
	}
}

func TestJSON_DefaultError(t *testing.T) {
	errResp := (&JSON{}).DefaultError().GetBody()

	if bytes.Compare(errResp, InternalServerErrorJsonBytes) != 0 {
		t.Error("Default error response invalid")
	}
}

func TestJSON_GetAcceptedType(t *testing.T) {
	at := (&JSON{}).GetAcceptedType()

	expected := "application/json"

	if at != expected {
		t.Errorf("Invalid value for accepted types, expected %s, got %s", expected, at)
	}
}

func TestJSON_Unmarshal(t *testing.T) {
	root := &JSON{}

	expectResponse(t, root.Unmarshal(nil), http.StatusNoContent, []byte(""))
	expectResponse(t, root.Unmarshal(true), -1, nil)
	expectResponse(t, root.Unmarshal(""), http.StatusNoContent, nil)
	expectResponse(t, root.Unmarshal("test"), http.StatusOK, []byte("test"))

	testJson := JSON{Code: http.StatusCreated, Body: "test"}
	expectResponse(t, root.Unmarshal(testJson), http.StatusCreated, []byte("test"))
	expectResponse(t, root.Unmarshal(&testJson), http.StatusCreated, []byte("test"))

	testJsonError := JSONError{Code: http.StatusMethodNotAllowed, Description: "test"}
	expectResponse(t, root.Unmarshal(testJsonError), http.StatusMethodNotAllowed, []byte("{\"code\":405,\"description\":\"test\"}"))
	expectResponse(t, root.Unmarshal(&testJsonError), http.StatusMethodNotAllowed, []byte("{\"code\":405,\"description\":\"test\"}"))
	testJsonError.Err = errors.New("testerror")
	expectResponse(t, root.Unmarshal(&testJsonError), http.StatusMethodNotAllowed, []byte("{\"code\":405,\"description\":\"test\"}"))

	expectResponse(t, root.Unmarshal(int(200)), http.StatusOK, []byte("")) // TODO Test No content on handle
	expectResponse(t, root.Unmarshal(int32(300)), http.StatusMultipleChoices, []byte(""))
	expectResponse(t, root.Unmarshal(int16(400)), http.StatusBadRequest, []byte(""))
	expectResponse(t, root.Unmarshal(int8(10)), 10, []byte(""))
	expectResponse(t, root.Unmarshal(uint16(402)), http.StatusPaymentRequired, []byte(""))
	expectResponse(t, root.Unmarshal(uint8(208)), http.StatusAlreadyReported, []byte(""))

	expectResponse(t, root.Unmarshal(errors.New("testerror")), http.StatusInternalServerError, []byte("{\"code\":500,\"description\":\"Internal Server Error\"}"))
	expectResponse(t, root.Unmarshal(CoderError(429)), http.StatusTooManyRequests, []byte("{\"code\":429,\"description\":\"Internal Server Error\"}"))
	expectResponse(t, root.Unmarshal(JSONMarshalError("jsonmarshalerror")), http.StatusInternalServerError, []byte("\"testerror: jsonmarshalerror\""))
	expectResponse(t, root.Unmarshal(JSONMarshalCoderError("jsonmarshalcodererror")), 21, []byte("\"testerror: jsonmarshalcodererror\""))

	// TODO map
	// TODO slice
	// TODO array
}

func TestJSON_GetContentType(t *testing.T) {
	root := &JSON{}
	expected := "application/json"
	got := root.GetContentType()

	if got != expected {
		t.Errorf("Invalid JSON content type, expected %s, got %s", expected, got)
	}
}

func TestJSON_MarshalJSON(t *testing.T) {
	root := &JSON{
		Code: http.StatusFailedDependency,
		Body: "testingmarshal",
	}

	bM, err := json.Marshal(root)
	bR, err := json.Marshal(root.Body)
	bMJ, err := root.MarshalJSON()

	if err != nil {
		t.Error(err)
	}

	if bytes.Compare(bM, bR) != 0 {
		t.Errorf("Expected marshal to equal, expected %s, got %s", bR, bM)
	}

	if bytes.Compare(bMJ, bR) != 0 {
		t.Errorf("Expected marshal to equal, expected %s, got %s", bR, bMJ)
	}
}

func TestJSON_GetCode(t *testing.T) {
	root := &JSON{
		Code: http.StatusLocked,
	}

	expected := http.StatusLocked
	got := root.GetCode()

	if got != expected {
		t.Errorf("Invalid code returned, expected %d, got %d", expected, got)
	}
}

func TestJSON_GetBody(t *testing.T) {
	root := &JSON{}

	var (
		expected []byte
		got      []byte
	)

	expected = []byte{}
	got = root.GetBody()

	if bytes.Compare(got, expected) != 0 {
		t.Errorf("Invalid body returned, expected %s, got %s", expected, got)
	}

	root.Body = []byte("testingbyteslice")
	expected = []byte("testingbyteslice")
	got = root.GetBody()

	if bytes.Compare(got, expected) != 0 {
		t.Errorf("Invalid body returned, expected %s, got %s", expected, got)
	}

	root.Body = "testingstring"
	expected = []byte("testingstring")
	got = root.GetBody()

	if bytes.Compare(got, expected) != 0 {
		t.Errorf("Invalid body returned, expected %s, got %s", expected, got)
	}

	root.Body = map[string]string{"key": "value"}
	expected = []byte("{\"key\":\"value\"}")
	got = root.GetBody()

	if bytes.Compare(got, expected) != 0 {
		t.Errorf("Invalid body returned, expected %s, got %s", expected, got)
	}

	root.Body = make(chan struct{}, 0)
	expected = InternalServerErrorJsonBytes
	got = root.GetBody()

	if bytes.Compare(got, expected) != 0 {
		t.Errorf("Invalid body returned, expected %s, got %s", expected, got)
	}
}

func TestJSON_Handle(t *testing.T) {
	root := &JSON{}

	ec := http.StatusInternalServerError
	eb := InternalServerErrorJsonBytes
	c, b := root.Handle()

	if c != ec {
		t.Errorf("Invalid code returned, expected %d, got %d", ec, c)
	}
	if bytes.Compare(b, eb) != 0 {
		t.Errorf("Invalid body returned, expected %s, got %s", eb, b)
	}

	root.Code = http.StatusLocked

	ec = http.StatusLocked
	eb = []byte{}
	c, b = root.Handle()

	if c != ec {
		t.Errorf("Invalid code returned, expected %d, got %d", ec, c)
	}
	if bytes.Compare(b, eb) != 0 {
		t.Errorf("Invalid body returned, expected %s, got %s", eb, b)
	}

	root.Code = http.StatusOK

	ec = http.StatusOK
	eb = []byte{}
	c, b = root.Handle()

	if c != ec {
		t.Errorf("Invalid code returned, expected %d, got %d", ec, c)
	}
	if bytes.Compare(b, eb) != 0 {
		t.Errorf("Invalid body returned, expected %s, got %s", eb, b)
	}

	root = &JSON{}
	root.Body = "testhandlebody"

	ec = http.StatusOK
	eb = []byte("testhandlebody")
	c, b = root.Handle()

	if c != ec {
		t.Errorf("Invalid code returned, expected %d, got %d", ec, c)
	}
	if bytes.Compare(b, eb) != 0 {
		t.Errorf("Invalid body returned, expected %s, got %s", eb, b)
	}

	root.Code = http.StatusOK
	root.Body = InternalServerErrorJsonBytes

	ec = http.StatusInternalServerError
	eb = InternalServerErrorJsonBytes
	c, b = root.Handle()

	if c != ec {
		t.Errorf("Invalid code returned, expected %d, got %d", ec, c)
	}
	if bytes.Compare(b, eb) != 0 {
		t.Errorf("Invalid body returned, expected %s, got %s", eb, b)
	}

	logMock := LogMock{Entry: logger.New("test").WithField("test", true)}
	root.log = logMock
	root.err = errors.New("testerr")

	_, _ = root.Handle()
	got := logMockCalled

	if got != 1 {
		t.Errorf("Log not called, expected %d, got %d", 1, got)
	}
}

func TestJSONError_GetCode(t *testing.T) {
	je := JSONError{
		Code: http.StatusTooEarly,
	}

	expected := http.StatusTooEarly
	got := je.GetCode()

	if got != expected {
		t.Errorf("Invalid JSONError code, expected %d, got %d", expected, got)
	}

	jePtr := &JSONError{
		Code: http.StatusUnauthorized,
	}

	expected = http.StatusUnauthorized
	got = jePtr.GetCode()

	if got != expected {
		t.Errorf("Invalid JSONError code, expected %d, got %d", expected, got)
	}
}

func expectResponse(t *testing.T, r Response, code int, body []byte) {
	t.Helper()

	if r == nil && body == nil && code == -1 {
		return
	} else if r == nil {
		t.Errorf("Invalid response, expected code %d and body %s, got nil", code, body)

		return
	}

	gotCode := r.GetCode()

	if gotCode != code {
		t.Errorf("Invalid response code, expected %d, got %d", code, r.GetCode())
	}

	gotBody := r.GetBody()

	if bytes.Compare(gotBody, body) != 0 {
		t.Errorf("Invalid response body, expected %s, got %s", body, gotBody)
	}
}
