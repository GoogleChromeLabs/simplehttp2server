`simplehttp2server` serves the current directory on an HTTP/2.0 capable server. This server is for development purposes only. `simplehttp2server` has partial, unofficial support for [Firebase’s JSON config]. Please see [disclaimer](#disclaimer) below.

## Binaries
`simplehttp2server` is `go get`-able:

```
$ go get github.com/GoogleChrome/simplehttp2server
```

Precompiled binaries can be found in the [release section](https://github.com/GoogleChrome/simplehttp2server/releases).

## Brew
You can also install `simplehttp2server` using brew if you are on macOS:

```
$ brew tap GoogleChrome/simplehttp2server https://github.com/GoogleChrome/simplehttp2server
$ brew install simplehttp2server
```

## Docker
If you have Docker set up, you can serve the current directory via `simplehttp2server` using the following command:

```
$ docker run -p 5000:5000 -v $PWD:/data surma/simplehttp2server
```

## Config

`simplehttp2server` has (partial) support for [Firebase’s JSON config]. This way you can add custom headers, rewrite rules and redirects. Here’s an example config:

```js
{
  "redirects": [
    {
      "source": "/send_me_somewhere",
      "destination": "https://google.com",
      "type": 301
    }
  ],
  "rewrites": [
    {
      "source": "/app/**",
      "destination": "/index.html"
    }
  ],
  "hosting": {
    "headers": [
      {
        "source": "**.html",
        "headers": [
          {
            "key": "Cache-Control",
            "value": "no-cache"
          },
          {
            "key": "Link",
            "value": "</header.jpg>; rel=preload; as=image"
          }
        ]
      }
    ]
  }
}
```

For details see the [Firebase’s documentation][Firebase’s JSON config].

## Disclaimer

I haven’t tested if the behavior of `simplehttp2server` _always_ matches the live server of Firebase, and some options (like `trailingSlash` and `cleanUrls`) are completely missing. Please open an issue if you find a discrepancy! The support is not offically endorsed by Firebase (yet 😜), so don’t rely on it!

## HTTP/2 PUSH

Any `Link` headers with `rel=preload` will be translated to a HTTP/2 PUSH, as is
common practice on static hosting platforms and CDNs. See the example above.

# License

Apache 2.

[Firebase’s JSON config]: https://firebase.google.com/docs/hosting/full-config
