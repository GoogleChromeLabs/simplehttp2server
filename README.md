`simplehttp2server` serves the current directory on an HTTP/2.0 capable server.
This server is for development purposes only.

# Push

All requests will be looked up in a file named `push.json`. If there is a key
for the request path, all resources in the array under that key will be pushed.

Example `push.json`:

```JS
{
  "/": [
    "/banner.jpg",
    "styles.css"
  ]
}
```

Support for weighting those pushes is not yet implemented.

# TLS Certificate

Since HTTP/2 requires TLS, `simplehttp2server` checks if `cert.pem` and
`key.pem` are present. If not, a self-signed certificate will be generated.

# Download

`simplehttp2server` is `go get`-able:

```
$ go get github.com/GoogleChrome/simplehttp2server
```

Precompiled binaries can be found in the [release section](https://github.com/GoogleChrome/simplehttp2server/releases).

# License

Apache 2.
