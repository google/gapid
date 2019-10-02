ereporter2/
===========

`ereporter2` is an embeddable modules that augments an AppEngine service with
native [logservice](https://cloud.google.com/appengine/docs/python/logs/) based
alerting.

It includes support for silencing based on signatures and client side exception
reporting. It exposes functionality to generate and email an hourly report via a
cron job.


### To use it in your service

  - Add to your `app.yaml` to enable the `/restricted/ereporter2/` endpoints:

```
includes:
- components/ereporter2
```

  - Add to your `cron.py`:

```
### ereporter2

- description: ereporter2 cleanup
  url: /internal/cron/ereporter2/cleanup
  schedule: every 1 hours

- description: ereporter2 mail exception report
  url: /internal/cron/ereporter2/mail
  schedule: every 1 hours synchronized
```

  - In your `main.py`, add:

```
from components import ereporter2

ereporter2.register_formatter()
```
