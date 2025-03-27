Service that just forwards incoming requests as-is, except for updating the http headers.
So mobile apps can indirectly make calls to an key authenticated API service without bundling any secret keys.

Where and how to forward is found in the incoming request headers:

- `X-Request-URL`, where to forward, value should be a full URL
- `X-Request-Method`, value should be GET, PUT or POST
- `X-Request-Header-[header key to append when forwarding]`, key to append and keyID of the value 

### Usage 

`./request_forwarder -p 9100 -s -h headers.json`
- `-p 9100` Sets the port to 9100
- `-s` Enables HTTPS mode. Will look for the certificate and key files named "server.crt" and "server.key" in the current directory. These files need to be present for the HTTPS server to start properly.
- `-h headers.json` Specifies the JSON file containing headers to add to forwarded requests. Other header key-values are preserved.

Sample `headers.json`
```json
{
  "key1": "abc"
  "key2": "xyz"
}
```

Now if the incoming request has `X-Request-Header-Authorization`:`key`, the forwarded request will append or replace the header `Authorization`: `abc`.

