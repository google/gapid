# endpoints_webapp2/

`endpoints_webapp2` is a package which implements Cloud Endpoints v1 over
webapp2 routes by acting as an adapter.

## Usage

The adapter is a drop-in replacement. Simply replace your calls to
`endpoints.api_server` with calls to `endpoints_webapp2.api_server`.
You will need to update your app configuration in as well.

### Before

```py
import endpoints

def get_routes():
  return endpoints.api_server([
    MyService,
    MyOtherService,
  ])
```

### After

```py
import endpoints_webapp2

def get_routes():
  return endpoints_webapp2.api_server([
    MyService,
    MyOtherService,
  ])
```

Using the adapter creates discovery routes for all your services. The default
base path is `/api` (cf. `/_ah/spi` on Cloud Endpoints v1 and `/_ah/api` on
Cloud Endpoints v2) so you'll need to update `app.yaml` accordingly. You can
change the base path by supplying a `base_path` keyword argument to
`endpoints_webapp2.api_server`.

```py
return endpoints_webapp2.api_server([MyService], base_path='/custom/path')
```
