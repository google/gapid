# endpoints_webapp2/templates/

[API explorer](https://apis-explorer.appspot.com/apis-explorer) creates a web UI
for an app's Cloud Endpoints services by hitting the app's `/static/proxy.html`
file under it's API base path (e.g. `/_ah/api/static/proxy.html` by default).

`proxy.html` is minified JavaScript which the API explorer uses to send
requests to the app's Cloud Endpoints, rather than loading the discovery
document directly then hitting the app's endpoints directly. For Cloud
Endpoints services, Apiary normally creates and ensures `proxy.html` can
be served.

The `templates` directory contains a copy of `proxy.html` with one change
allowing the base path to be customized. It is intended to be served at
a static location by the adapter so that the API explorer can function.
