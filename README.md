# Request Forwarder

A very lightweight proxy service that forwards incoming requests while adding necessary authentication headers. So mobile apps can make API calls without bundling sensitive credentials.

The forwarding details are extracted from specific headers in the incoming request:

- `X-Request-URL`: The complete URL where the request should be forwarded
- `X-Request-Method`: The HTTP method to use (GET, PUT, or POST)
- `X-Request-Key-[header_name]`: Maps to a keyID in the configuration file

## Usage

```
./request_forwarder -p 9100 -s -h headers.json
```

### Options

- `-p PORT`: Port to listen on (default: 9100)
- `-s`: Enable HTTPS (requires server.crt and server.key in the current directory)
- `-h FILE`: Path to JSON file containing key-value mappings

- `-k SEED`: Enable validating incoming requests and use SEED for the key validation. The key is a combo of the seed, date (updated daily) all CRC32'd, see the code

The headers file is a simple JSON dictionary mapping keyIDs to their values:

```json
{
  "key1": "abc",
  "key2": "xyz"
}
```

## Example

If your headers.json contains `{"api_key1": "1234567890"}` and the incoming request includes:
```
X-Request-URL: https://api.example-service.com/data
X-Request-Method: GET
X-Request-Key-Authorization: api_key1
```

The service will forward a GET request to https://api.example-service.com/data with the header `Authorization: 1234567890` added.

All other original headers are preserved in the forwarded request.
