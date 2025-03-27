Service that just forwards incoming requests as-is, except for updating the http headers.
So mobile apps can indirectly make calls to an key authenticated API service without bundling any secret keys.

### Usage 

`./request_forwarder -p 9100 -s -h headers.json`
- `-p 9100` Sets the port to 9100
- `-s` Enables HTTPS mode. Will look for the certificate and key files named "server.crt" and "server.key" in the current directory. These files need to be present for the HTTPS server to start properly.
- `-h headers.json` Specifies the JSON file containing headers to add to forwarded requests. Other header key-values are preserved.

Sample `headers.json`
```json
{
  "headers": {
    "Content-Type": "application/json",
    "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "X-API-Key": "your-secret-api-key",
    "X-Service-Client": "mobile-proxy"
  }
}
```
