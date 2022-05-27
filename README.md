# caddy-middleware
## Build
```
xcaddy build --output ./caddy.exe --with github.com/liornabat/hocoos-middleware
```
## Example
````json
{
  "logging": {
    "logs": {
      "default": {
        "level": "DEBUG"
      }
    }
  },
  "apps": {
    "http": {
      "servers": {
        "myserver": {
          "listen": [":443",":80"],
          "routes": [
            {
              "match": [{"path": ["/*"]}],
              "handle": [
                {
                  "handler": "hocoos_middleware",
                  "redis_url": "redis://username:password@localhost:6379/0"
                },
                {
                "handler": "static_response",
                "status_code": "",
                "body": "Hi there!",
                "close": false,
                "abort": false
                }
              ]
            }
          ]
        }
      }
    }
  }
}

````
