# caddy-middleware
## Build
```
xcaddy build --output ./caddy.exe --with github.com/liornabat/hocoos-middleware@v0.1.0
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
                  "redis_url": "redis://username:password@localhost:6379/0",
                  "cache_ttl": 60,
                  "exclude_hosts": "hocoos.cafe,hocoos.com,localhost"
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
