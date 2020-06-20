# Response writer
Middleware to make handlers more convenient.

This package removes the need to pass the writer to the handler.
It consolidates the request and params. Resulting in the need for only one argument. 

Any response should be returned in one of many supported formats.

Any response can support one or more response types. And depending on the returned format, handle some or all of them.

# Install
```bash
go get -u "peterdekok.nl/gotools/responsewriter"
```

```golang
import "peterdekok.nl/gotools/responsewriter"
```
