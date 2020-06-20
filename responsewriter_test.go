package responsewriter

import (
	"errors"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
	"net/http"
	"peterdekok.nl/gotools/logger"
	"peterdekok.nl/gotools/responsewriter/responsetype"
	"testing"
)

type LogMock struct {
	*logrus.Entry
}

var logMockCalled int

func (l LogMock) WithFields(fields logrus.Fields) *logrus.Entry {
	logMockCalled++

	return l.Entry.WithFields(fields)
}

type mockHandler struct {
	called int

	i interface{}
}

func (mh *mockHandler) fn(_ *Request) interface{} {
	mh.called++

	return mh.i
}

type mockResponseType struct {
	t       string
	resp    responsetype.Response
	defResp responsetype.Response
}

func (mrt mockResponseType) GetAcceptedType() string                       { return mrt.t }
func (mrt mockResponseType) String() string                                { return mrt.t }
func (mrt mockResponseType) Unmarshal(_ interface{}) responsetype.Response { return mrt.resp }
func (mrt mockResponseType) DefaultError() responsetype.Response           { return mrt.defResp }

type hoBag struct {
	h        http.Header
	b        [][]byte
	whcalled []int
}
type headerOnlyResponseWriter struct {
	bag *hoBag
}

func (ho headerOnlyResponseWriter) Header() http.Header { return ho.bag.h }
func (ho headerOnlyResponseWriter) Write(b []byte) (int, error) {
	ho.bag.b = append(ho.bag.b, b)

	return 0, errors.New("write failed intentional")
}
func (ho headerOnlyResponseWriter) WriteHeader(c int) { ho.bag.whcalled = append(ho.bag.whcalled, c) }

type mockResponse struct {
	code int
	body []byte
	ctt  string
}

func (mr *mockResponse) Handle() (int, []byte)  { return mr.code, mr.body }
func (mr *mockResponse) GetBody() []byte        { panic("NOIMPL") }
func (mr *mockResponse) GetContentType() string { return mr.ctt }
func (mr *mockResponse) GetCode() int           { panic("NOIMPL") }

func TestNewRequest(t *testing.T) {
	r := &http.Request{Method: "TESTMETHOD"}
	p := httprouter.Params{httprouter.Param{Key: "testparam", Value: "testvalue"}}

	req := NewRequest(r, p)

	emet := "TESTMETHOD"
	met := req.Method

	if met != emet {
		t.Errorf("Unexpected method, expected %s, got %s", emet, met)
	}

	if req.Params == nil {
		t.Error("Expected params to be defined, got nil")
	}

	epv := "testvalue"
	pv := req.Params.ByName("testparam")

	if pv != epv {
		t.Errorf("Unexpected param value, expected %s, got %s", epv, pv)
	}
}

func TestResponseHandler(t *testing.T) {
	logMock := LogMock{Entry: logger.New("test").WithField("test", true)}
	log = logMock

	mh := &mockHandler{}

	func() {
		defer func() {
			expected := "Invalid response type given for response handler"

			if err := recover(); err == nil {
				t.Error("Expected panic")
			} else if err != expected {
				t.Errorf("Expected panic to be: %s, got %s", expected, err)
			}
		}()

		ResponseHandler(mh.fn, nil)
	}()

	mrt := &mockResponseType{}

	func() {
		defer func() {
			expected := "Invalid accepted response type given"

			if err := recover(); err == nil {
				t.Error("Expected panic")
			} else if err != expected {
				t.Errorf("Expected panic to be: %s, got %s", expected, err)
			}
		}()

		ResponseHandler(mh.fn, mrt)
	}()

	mrtA := &mockResponseType{t: "first/content-type", resp: &mockResponse{code: 101, body: []byte("noempty-first"), ctt: "first/resp-content-type"}}
	mrtB := &mockResponseType{t: "second/content-type", resp: &mockResponse{code: 102, body: []byte("noempty-second")}}
	mrtC := &mockResponseType{t: "third/content-type", resp: nil, defResp: &mockResponse{code: 200, body: nil, ctt: "third/resp-content-type"}}

	fn := ResponseHandler(mh.fn, mrtA, mrtB, mrtC)

	w := headerOnlyResponseWriter{
		bag: &hoBag{
			h:        make(http.Header),
			b:        make([][]byte, 0),
			whcalled: make([]int, 0),
		},
	}
	r := &http.Request{Header: http.Header{"Accept": []string{"first/content-type"}}}
	p := httprouter.Params{}

	//
	// FIRST RUN
	//
	fn(w, r, p)

	emhc := 1
	mhc := mh.called

	if mhc != emhc {
		t.Errorf("Invalid handler call count, expected %d, got %d", emhc, mhc)
	}

	eah := "first/resp-content-type"
	ah := w.Header().Get("Content-Type")

	if ah != eah {
		t.Errorf("Invalid set header call, expected %s, got %s", eah, ah)
	}

	ewhl := 1
	whl := len(w.bag.whcalled)

	if whl != ewhl {
		t.Errorf("Invalid write header called count, expected %d, got %d", ewhl, whl)
	}

	ewhc := 101
	whc := w.bag.whcalled[0]

	if whc != ewhc {
		t.Errorf("Invalid write header called code, expected %d, got %d", ewhc, whc)
	}

	elmc := 1
	lmc := logMockCalled

	if lmc != elmc {
		t.Errorf("Log not called, expected %d, got %d", elmc, lmc)
	}

	//
	// SECOND RUN
	//
	r.Header.Set("Accept", "second/content-type")

	w.Header().Del("Content-Type")
	fn(w, r, p)

	emhc = 2
	mhc = mh.called

	if mhc != emhc {
		t.Errorf("Invalid handler call count, expected %d, got %d", emhc, mhc)
	}

	eah = "text/plain"
	ah = w.Header().Get("Content-Type")

	if ah != eah {
		t.Errorf("Invalid set header call, expected %s, got %s", eah, ah)
	}

	ewhl = 2
	whl = len(w.bag.whcalled)

	if whl != ewhl {
		t.Errorf("Invalid write header called count, expected %d, got %d", ewhl, whl)
	}

	ewhc = 102
	whc = w.bag.whcalled[1]

	if whc != ewhc {
		t.Errorf("Invalid write header called code, expected %d, got %d", ewhc, whc)
	}

	elmc = 2
	lmc = logMockCalled

	if lmc != elmc {
		t.Errorf("Log not called, expected %d, got %d", elmc, lmc)
	}

	//
	// THIRD RUN
	//
	r.Header.Set("Accept", "third/content-type")

	w.Header().Del("Content-Type")
	fn(w, r, p)

	emhc = 3
	mhc = mh.called

	if mhc != emhc {
		t.Errorf("Invalid handler call count, expected %d, got %d", emhc, mhc)
	}

	eah = ""
	ah = w.Header().Get("Content-Type")

	if ah != eah {
		t.Errorf("Invalid set header call, expected %s, got %s", eah, ah)
	}

	ewhl = 3
	whl = len(w.bag.whcalled)

	if whl != ewhl {
		t.Errorf("Invalid write header called count, expected %d, got %d", ewhl, whl)
	}

	ewhc = 204
	whc = w.bag.whcalled[2]

	if whc != ewhc {
		t.Errorf("Invalid write header called code, expected %d, got %d", ewhc, whc)
	}

	elmc = 2
	lmc = logMockCalled

	if lmc != elmc {
		t.Errorf("Log not called, expected %d, got %d", elmc, lmc)
	}

}
