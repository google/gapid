auth/
=====

`auth` is a library that provides authorization and authentication functionality
for webapp2 and Cloud Endpoints apps. Acts as a client for
[auth_service](../../../auth_service).

### To use it in your service

  - Add to your `app.yaml` to enable the `/auth` end points:

```
includes:
- components/auth
- components/static_third_party.yaml

libraries:
- name: endpoints
  version: "1.0"
- name: jinja2
  version: "2.6"
- name: pycrypto
  version: "2.6"
- name: webapp2
  version: "2.5.2"
- name: webob
  version: "1.2.3"
```

  - Add to your `main.py`:

```
BASE_DIR = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(BASE_DIR, 'components', 'third_party'))
```

  - In your `acl.py`, implement your group ACL implementation by leveraging
    `components.auth`.
  - Inherit webapp2 handlers from `auth.AuthenticatingHandler`.
  - Decorate webapp2 handlers with `@auth.require(acl.is_foo)` to enforce the
    ACLs.
  - Use `@auth.endpoints_api` instead of `@endpoints.api`, use
    `@auth.endpoints_method` instead of `@endpoints.method`.
  - All POST\PUT\DELETE handlers must be aware of XSRF token, e.g. put it in
    `xsrf_token` hidden field.
