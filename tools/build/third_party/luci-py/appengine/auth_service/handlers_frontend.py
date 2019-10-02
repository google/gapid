# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""This module defines Auth Server frontend url handlers."""

import os
import base64

import webapp2

from google.appengine.api import app_identity

from components import auth
from components import template
from components import utils

from components.auth import model
from components.auth import tokens
from components.auth import version
from components.auth.proto import replication_pb2
from components.auth.ui import rest_api
from components.auth.ui import ui

import acl
import config
import importer
import pubsub
import replication


# Path to search for jinja templates.
TEMPLATES_DIR = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), 'templates')


################################################################################
## UI handlers.


class WarmupHandler(webapp2.RequestHandler):
  def get(self):
    auth.warmup()
    self.response.headers['Content-Type'] = 'text/plain; charset=utf-8'
    self.response.write('ok')


class EmailHandler(webapp2.RequestHandler):
  """Blackhole any email sent."""
  def post(self, to):
    pass


class ConfigHandler(ui.UINavbarTabHandler):
  """Page with simple UI for service-global configuration."""
  navbar_tab_url = '/auth/config'
  navbar_tab_id = 'config'
  navbar_tab_title = 'Config'
  # config.js here won't work because there's global JS var 'config' already.
  js_file_url = '/auth_service/static/js/config_page.js'
  template_file = 'auth_service/config.html'


class ServicesHandler(ui.UINavbarTabHandler):
  """Page with management UI for linking services."""
  navbar_tab_url = '/auth/services'
  navbar_tab_id = 'services'
  navbar_tab_title = 'Services'
  js_file_url = '/auth_service/static/js/services.js'
  template_file = 'auth_service/services.html'


def get_additional_ui_data():
  """Gets injected into Jinja and Javascript environment."""
  if not config.is_remote_configured():
    return {'auth_service_config_locked': False}
  config_revisions = {}
  for path, rev in config.get_revisions().iteritems():
    config_revisions[path] = {
      'rev': rev.revision if rev else 'none',
      'url': rev.url if rev else 'about:blank',
    }
  return {
    'auth_service_config_locked': True,
    'auth_service_configs': {
      'remote_url': config.get_remote_url(),
      'revisions': config_revisions,
    },
  }


################################################################################
## API handlers.


class LinkTicketToken(auth.TokenKind):
  """Parameters for ServiceLinkTicket.ticket token."""
  expiration_sec = 24 * 3600
  secret_key = auth.SecretKey('link_ticket_token')
  version = 1


class AuthDBRevisionsHandler(auth.ApiHandler):
  """Serves deflated AuthDB proto message with snapshot of all groups.

  Args:
    rev: version of the snapshot to get ('latest' or concrete revision number).
        Not all versions may be available (i.e. there may be gaps in revision
        numbers).
    skip_body: if '1' will not return actual snapshot, just its SHA256 hash,
        revision number and timestamp.
  """

  @auth.require(lambda: (
      auth.is_admin() or
      acl.is_trusted_service() or
      replication.is_replica(auth.get_current_identity())))
  def get(self, rev):
    skip_body = self.request.get('skip_body') == '1'
    if rev == 'latest':
      snapshot = replication.get_latest_auth_db_snapshot(skip_body)
    else:
      try:
        rev = int(rev)
      except ValueError:
        self.abort_with_error(400, text='Bad revision number, not an integer')
      snapshot = replication.get_auth_db_snapshot(rev, skip_body)
    if not snapshot:
      self.abort_with_error(404, text='No such snapshot: %s' % rev)
    resp = {
      'auth_db_rev': snapshot.key.integer_id(),
      'created_ts': utils.datetime_to_timestamp(snapshot.created_ts),
      'sha256': snapshot.auth_db_sha256,
    }
    if not skip_body:
      assert snapshot.auth_db_deflated
      resp['deflated_body'] = base64.b64encode(snapshot.auth_db_deflated)
    self.send_response({'snapshot': resp})


class AuthDBSubscriptionAuthHandler(auth.ApiHandler):
  """Manages authorization to PubSub topic for AuthDB change notifications.

  Members of 'auth-trusted-services' group may use this endpoint to make sure
  they can attach subscriptions to AuthDB change notification stream.
  """

  def subscriber_email(self):
    """Validates caller is using email for auth, returns it.

    Raises HTTP 400 if some other kind of authentication is used. Only emails
    are supported by PubSub.
    """
    caller = auth.get_current_identity()
    if not caller.is_user:
      self.abort_with_error(400, text='Caller must use email-based auth')
    return caller.name

  @auth.require(acl.is_trusted_service)
  def get(self):
    """Queries whether the caller is authorized to attach subscriptions already.

    Response body:
    {
      'topic': <full name of PubSub topic with AuthDB change notifications>,
      'authorized': <boolean>
    }
    """
    try:
      return self.send_response({
        'topic': pubsub.topic_name(),
        'authorized': pubsub.is_authorized_subscriber(self.subscriber_email()),
      })
    except pubsub.Error as e:
      self.abort_with_error(409, text=str(e))

  @auth.require(acl.is_trusted_service)
  def post(self):
    """Grants caller "pubsub.subscriber" role on change notifications topic.

    Response body:
    {
      'topic': <full name of PubSub topic with AuthDB change notifications>,
      'authorized': true
    }
    """
    try:
      pubsub.authorize_subscriber(self.subscriber_email())
      return self.send_response({
        'topic': pubsub.topic_name(),
        'authorized': True,
      })
    except pubsub.Error as e:
      self.abort_with_error(409, text=str(e))

  @auth.require(acl.is_trusted_service)
  def delete(self):
    """Revokes authorization if it exists.

    Response body:
    {
      'topic': <full name of PubSub topic with AuthDB change notifications>,
      'authorized': false
    }
    """
    try:
      pubsub.deauthorize_subscriber(self.subscriber_email())
      return self.send_response({
        'topic': pubsub.topic_name(),
        'authorized': False,
      })
    except pubsub.Error as e:
      self.abort_with_error(409, text=str(e))


class ImporterConfigHandler(auth.ApiHandler):
  """Reads and sets configuration of the group importer."""

  @auth.require(acl.has_access)
  def get(self):
    self.send_response({'config': importer.read_config()})

  @auth.require(auth.is_admin)
  def post(self):
    if config.is_remote_configured():
      self.abort_with_error(409, text='The configuration is managed elsewhere')
    try:
      importer.write_config(
          text=self.parse_body().get('config'),
          modified_by=auth.get_current_identity())
    except ValueError as ex:
      self.abort_with_error(400, text=str(ex))
    self.send_response({'ok': True})


class ImporterIngestTarballHandler(auth.ApiHandler):
  """Accepts PUT with a tarball containing a bunch of groups to import.

  The request body is expected to be the tarball as a raw byte stream.

  See proto/config.proto, GroupImporterConfig for more details.
  """

  # For some reason webapp2 attempts to deserialize the body as a form data when
  # searching for XSRF token (which doesn't work when the body is tarball).
  # Disable this (along with the cookies-based auth, we want only OAuth2).
  xsrf_token_request_param = None
  xsrf_token_enforce_on = ()

  @classmethod
  def get_auth_methods(cls, conf):
    return [auth.oauth_authentication]

  # The real authorization check is inside 'ingest_tarball'. This one just
  # rejects anonymous calls earlier.
  @auth.require(lambda: not auth.get_current_identity().is_anonymous)
  def put(self, name):
    try:
      groups, auth_db_rev = importer.ingest_tarball(name, self.request.body)
      self.send_response({
        'groups': groups,
        'auth_db_rev': auth_db_rev,
      })
    except importer.BundleImportError as e:
      self.abort_with_error(400, error=str(e))


class ServiceListingHandler(auth.ApiHandler):
  """Lists registered replicas with their state."""

  @auth.require(acl.has_access)
  def get(self):
    services = sorted(
        replication.AuthReplicaState.query(
            ancestor=replication.replicas_root_key()),
        key=lambda x: x.key.id())
    last_auth_state = model.get_replication_state()
    self.send_response({
      'services': [
        x.to_serializable_dict(with_id_as='app_id') for x in services
      ],
      'auth_code_version': version.__version__,
      'auth_db_rev': {
        'primary_id': last_auth_state.primary_id,
        'rev': last_auth_state.auth_db_rev,
        'ts': utils.datetime_to_timestamp(last_auth_state.modified_ts),
      },
      'now': utils.datetime_to_timestamp(utils.utcnow()),
    })


class GenerateLinkingURL(auth.ApiHandler):
  """Generates an URL that can be used to link a new replica.

  See auth/proto/replication.proto for the description of the protocol.
  """

  @auth.require(auth.is_admin)
  def post(self, app_id):
    # On local dev server |app_id| may use @localhost:8080 to specify where
    # app is running.
    custom_host = None
    if utils.is_local_dev_server():
      app_id, _, custom_host = app_id.partition('@')

    # Generate an opaque ticket that would be passed back to /link_replica.
    # /link_replica will verify HMAC tag and will ensure the request came from
    # application with ID |app_id|.
    ticket = LinkTicketToken.generate([], {'app_id': app_id})

    # ServiceLinkTicket contains information that is needed for Replica
    # to figure out how to contact Primary.
    link_msg = replication_pb2.ServiceLinkTicket()
    link_msg.primary_id = app_identity.get_application_id()
    link_msg.primary_url = self.request.host_url
    link_msg.generated_by = auth.get_current_identity().to_bytes()
    link_msg.ticket = ticket

    # Special case for dev server to simplify local development.
    if custom_host:
      assert utils.is_local_dev_server()
      host = 'http://%s' % custom_host
    else:
      # Use same domain as auth_service. Usually it's just appspot.com.
      current_hostname = app_identity.get_default_version_hostname()
      domain = current_hostname.partition('.')[2]
      naked_app_id = app_id
      if ':' in app_id:
        naked_app_id = app_id[app_id.find(':')+1:]
      host = 'https://%s.%s' % (naked_app_id, domain)

    # URL to a handler on Replica that initiates Replica <-> Primary handshake.
    url = '%s/auth/link?t=%s' % (
        host, tokens.base64_encode(link_msg.SerializeToString()))
    self.send_response({'url': url}, http_code=201)


class LinkRequestHandler(auth.AuthenticatingHandler):
  """Called by a service that wants to become a Replica."""

  # Handler uses X-Appengine-Inbound-Appid header protected by GAE.
  xsrf_token_enforce_on = ()

  def reply(self, status):
    """Sends serialized ServiceLinkResponse as a response."""
    msg = replication_pb2.ServiceLinkResponse()
    msg.status = status
    self.response.headers['Content-Type'] = 'application/octet-stream'
    self.response.write(msg.SerializeToString())

  # Check that the request came from some GAE app. It filters out most requests
  # from script kiddies right away.
  @auth.require(lambda: auth.get_current_identity().is_service)
  def post(self):
    # Deserialize the body. Dying here with 500 is ok, it should not happen, so
    # if it is happening, it's nice to get an exception report.
    request = replication_pb2.ServiceLinkRequest.FromString(self.request.body)

    # Ensure the ticket was generated by us (by checking HMAC tag).
    ticket_data = None
    try:
      ticket_data = LinkTicketToken.validate(request.ticket, [])
    except tokens.InvalidTokenError:
      self.reply(replication_pb2.ServiceLinkResponse.BAD_TICKET)
      return

    # Ensure the ticket was generated for the calling application.
    replica_app_id = ticket_data['app_id']
    expected_ident = auth.Identity(auth.IDENTITY_SERVICE, replica_app_id)
    if auth.get_current_identity() != expected_ident:
      self.reply(replication_pb2.ServiceLinkResponse.AUTH_ERROR)
      return

    # Register the replica. If it is already there, will reset its known state.
    replication.register_replica(replica_app_id, request.replica_url)
    self.reply(replication_pb2.ServiceLinkResponse.SUCCESS)


################################################################################
## Application routing boilerplate.


def get_routes():
  # Use special syntax on dev server to specify where app is running.
  app_id_re = r'[0-9a-zA-Z_\-\:\.]*'
  if utils.is_local_dev_server():
    app_id_re += r'(@localhost:[0-9]+)?'

  # Auth service extends the basic UI and API provided by Auth component.
  routes = []
  routes.extend(rest_api.get_rest_api_routes())
  routes.extend(ui.get_ui_routes())
  routes.extend([
    # UI routes.
    webapp2.Route(
        r'/', webapp2.RedirectHandler, defaults={'_uri': '/auth/groups'}),
    webapp2.Route(r'/_ah/mail/<to:.+>', EmailHandler),
    webapp2.Route(r'/_ah/warmup', WarmupHandler),

    # API routes.
    webapp2.Route(
        r'/auth_service/api/v1/authdb/revisions/<rev:(latest|[0-9]+)>',
        AuthDBRevisionsHandler),
    webapp2.Route(
        r'/auth_service/api/v1/authdb/subscription/authorization',
        AuthDBSubscriptionAuthHandler),
    webapp2.Route(
        r'/auth_service/api/v1/importer/config',
        ImporterConfigHandler),
    webapp2.Route(
        r'/auth_service/api/v1/importer/ingest_tarball/<name:.+>',
        ImporterIngestTarballHandler),
    webapp2.Route(
        r'/auth_service/api/v1/internal/link_replica',
        LinkRequestHandler),
    webapp2.Route(
        r'/auth_service/api/v1/services',
        ServiceListingHandler),
    webapp2.Route(
        r'/auth_service/api/v1/services/<app_id:%s>/linking_url' % app_id_re,
        GenerateLinkingURL),
  ])
  return routes


def create_application(debug):
  replication.configure_as_primary()
  rest_api.set_config_locked(config.is_remote_configured)

  # Configure UI appearance, add all custom tabs.
  ui.configure_ui(
      app_name='Auth Service',
      ui_tabs=[
        ui.GroupsHandler,
        ui.ChangeLogHandler,
        ui.LookupHandler,
        ServicesHandler,
        ui.OAuthConfigHandler,
        ui.IPWhitelistsHandler,
        ConfigHandler,
        ui.ApiDocHandler,
      ],
      ui_data_callback=get_additional_ui_data)
  template.bootstrap({'auth_service': TEMPLATES_DIR})

  # Add a fake admin for local dev server.
  if utils.is_local_dev_server():
    auth.bootstrap_group(
        auth.ADMIN_GROUP,
        [auth.Identity(auth.IDENTITY_USER, 'test@example.com')],
        'Users that can manage groups')
  return webapp2.WSGIApplication(get_routes(), debug=debug)
