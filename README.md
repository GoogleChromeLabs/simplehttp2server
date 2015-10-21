`SimpleHttp2Server` serves the current directory on an HTTP/2.0 capable server.

All requests will be looked up in a file named `push.json`. If there is a key
for the request path, all resources in the array under that key will be pushed
(see demo directory).

Example `push.json`:

```JS
{
  "/": [
    "/banner.jpg",
    "styles.css"
  ]
}
```

If no certificate files (`cert.pem` and `key.pem`) are present,
a self-signed certificate will be generated.
