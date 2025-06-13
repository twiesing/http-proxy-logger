# HTTP proxy logger
This is HTTP proxy which prints HTTP requests and responses to the console,
including their bodies. Responses compressed with `gzip` or `deflate` are
automatically decompressed in the logs for readability.
For example:
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
## Build image
This project is built with **Go 1.23**.

`docker build -t stn1slv/http-proxy-logger .`
## Start
`docker run --rm -it -p 8888:8888 -e PORT=8888 -e TARGET=http://demo7704619.mockable.io stn1slv/http-proxy-logger`
