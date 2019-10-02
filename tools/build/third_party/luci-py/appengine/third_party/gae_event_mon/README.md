# Setting up event monitoring on App Engine.

1.  Symlink `gae_ts_mon` and `gae_event_mon` into your appengine app. If you are
    using timeseries monitoring, you will already have gae_ts_mon symlinked.

        cd infra/appengine/myapp
        ln -s ../../appengine_module/gae_ts_mon .
        ln -s ../../appengine_module/gae_event_mon .

1.  Initialize the library in your request handler.

        import gae_event_mon

        [...]

        gae_event_mon.initialize('service_name')

    You must do this in every top-level request handler that's listed in your
    app.yaml to ensure metrics are registered no matter which type of request
    an instance receives first.

You're done! You can now use `event_mon` exactly as you normally would using the
`infra_libs.event_mon` module. Here's a quick example:

    from infra_libs import event_mon

    class MyHandler(webapp2.RequestHandler):
      def get(self):
        count = goat_teleporter.teleport()

        event = event_mon.Event('POINT')
        event.proto.goat_teleported_event.num_goats = count
        event.send()

        self.response.write('Teleported %d goats this time' % count)


## Appengine Modules

Multiple Appengine modules are fully supported - the module name will appear as
part of the `event_source.host_name` field in the following format:
"`module_name`, version".

[chrome-infra-mon-pubsub project]: https://console.developers.google.com/project/chrome-infra-mon-pubsub/cloudpubsub/topicList
