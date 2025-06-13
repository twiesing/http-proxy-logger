# HTTP Proxy Logger

HTTP Proxy Logger is a small reverse proxy that prints incoming HTTP requests
and outgoing responses to stdout. Bodies compressed with `gzip` or `deflate`
are automatically decompressed in the logs so that you can easily inspect them.
The output uses ANSI colors similar to `HTTPie`: request and response lines,
header names, and JSON or XML bodies are highlighted for readability.

## Example output

```
2021/05/05 03:50:44 ---REQUEST 3---

POST /mocking/contacts HTTP/1.1
Host: demo7704619.mockable.io
User-Agent: PostmanRuntime/7.28.0
Content-Length: 63
Accept: */*
Accept-Encoding: gzip, deflate, br
Cache-Control: no-cache
Content-Type: application/json
X-Forwarded-For: 172.17.0.1

{
    "firstName": "Stanislav",
    "lastName": "Deviatov"
}

2021/05/05 03:50:44 ---RESPONSE 3---

HTTP/1.1 201 Created
Content-Length: 68
Access-Control-Allow-Origin: *
Content-Type: application/json; charset=UTF-8
Date: Wed, 05 May 2021 03:50:45 GMT
Server: Google Frontend
X-Cloud-Trace-Context: 83ac5937ae7ba8f3ef96ee941227b1b0

{
  "salesforceId": "a0C3L0000008ZSNUA2",
  "action": "updated"
}
```

## Building

The project requires **Go 1.23**.

### Build a binary

```bash
go build -o http-proxy-logger
```

### Build a Docker image

```bash
docker build -t stn1slv/http-proxy-logger .
```

## Running

Set the `TARGET` environment variable to the upstream server and optionally
`PORT` for the listen address. You can run the proxy either directly or inside
a Docker container.

### Local execution

```bash
TARGET=http://example.com PORT=8888 ./http-proxy-logger
```

### Docker

```bash
docker run --rm -it -p 8888:8888 \
  -e PORT=8888 \
  -e TARGET=http://demo7704619.mockable.io \
  stn1slv/http-proxy-logger
```

The proxy will forward traffic to the target and log each request/response pair
using the format shown above.

## License

This project is licensed under the MIT License.
