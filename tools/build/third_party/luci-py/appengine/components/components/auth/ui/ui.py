# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Auth management UI handlers."""

import functools
import json
import os
import re
import webapp2

from components import template
from components import utils

from . import acl
from . import rest_api

from .. import api
from .. import change_log
from .. import handler
from .. import model
from .. import replication


# templates/.
TEMPLATES_DIR = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), 'templates')


# Global static configuration set in 'configure_ui'.
_ui_app_name = 'Unknown'
_ui_data_callback = None
_ui_navbar_tabs = ()


def configure_ui(app_name, ui_tabs=None, ui_data_callback=None):
  """Modifies global configuration of Auth UI.

  Args:
    app_name: name of the service (visible in page headers, titles, etc.)
    ui_tabs: list of UINavbarTabHandler subclasses that define tabs to show, or
        None to show the standard set of tabs.
    ui_data_callback: an argumentless callable that returns a dict with
        additional data to return to authenticated users. It can be used by
        server and client side code to render templates. Used by auth_service.
  """
  global _ui_app_name
  global _ui_data_callback
  global _ui_navbar_tabs
  _ui_app_name = app_name
  _ui_data_callback = ui_data_callback
  if ui_tabs is not None:
    assert all(issubclass(cls, UINavbarTabHandler) for cls in ui_tabs)
    _ui_navbar_tabs = tuple(ui_tabs)
  template.bootstrap({'auth': TEMPLATES_DIR})


def get_ui_routes():
  """Returns a list of routes with auth UI handlers."""
  routes = []
  if not utils.should_disable_ui_routes():
    # Routes for registered navbar tabs.
    for cls in _ui_navbar_tabs:
      routes.extend(cls.get_webapp2_routes())
    # Routes for everything else.
    routes.extend([
      webapp2.Route(r'/auth', MainHandler),
      webapp2.Route(r'/auth/bootstrap', BootstrapHandler, name='bootstrap'),
      webapp2.Route(r'/auth/bootstrap/oauth', BootstrapOAuthHandler),
      webapp2.Route(r'/auth/link', LinkToPrimaryHandler),
      webapp2.Route(r'/auth/listing', GroupListingHandler),
    ])
  return routes


def forbid_ui_on_replica(method):
  """Decorator for methods that are not allowed to be called on Replica.

  If such method is called on a service in Replica mode, it would return
  HTTP 405 "Method Not Allowed".
  """
  @functools.wraps(method)
  def wrapper(self, *args, **kwargs):
    assert isinstance(self, webapp2.RequestHandler)
    if model.is_replica():
      primary_url = model.get_replication_state().primary_url
      self.abort(
          405,
          detail='Not allowed on a replica, see primary at %s' % primary_url)
    return method(self, *args, **kwargs)
  return wrapper


def redirect_ui_on_replica(method):
  """Decorator for methods that redirect to Primary when called on replica.

  If such method is called on a service in Replica mode, it would return
  HTTP 302 redirect to corresponding method on Primary.
  """
  @functools.wraps(method)
  def wrapper(self, *args, **kwargs):
    assert isinstance(self, webapp2.RequestHandler)
    assert self.request.method == 'GET'
    if model.is_replica():
      primary_url = model.get_replication_state().primary_url
      protocol = 'http://' if utils.is_local_dev_server() else 'https://'
      assert primary_url and primary_url.startswith(protocol), primary_url
      assert self.request.path_qs.startswith('/'), self.request.path_qs
      self.redirect(primary_url.rstrip('/') + self.request.path_qs, abort=True)
    return method(self, *args, **kwargs)
  return wrapper


################################################################################
## Admin routes. The use cookies and GAE's "is_current_user_admin" for authn.


class AdminPageHandler(handler.AuthenticatingHandler):
  """Base class for handlers involved in bootstrap processes."""

  # TODO(vadimsh): Enable CSP nonce for styles too. We'll need to get rid of
  # all 'style=...' attributes first.
  csp_use_script_nonce = True

  @classmethod
  def get_auth_methods(cls, conf):
    # This method sets 'is_superuser' bit for GAE-level admins.
    return [handler.gae_cookie_authentication]

  def reply(self, path, env=None, status=200):
    """Render template |path| to response using given environment.

    Args:
      path: path to a template, relative to templates/.
      env: additional environment dict to use when rendering the template.
      status: HTTP status code to return.
    """
    full_env = {
      'app_name': _ui_app_name,
      'csp_nonce': self.csp_nonce,
      'identity': api.get_current_identity(),
      'logout_url': json.dumps(self.create_logout_url('/')), # see base.html
      'xsrf_token': self.generate_xsrf_token(),
    }
    full_env.update(env or {})
    self.response.set_status(status)
    self.response.headers['Content-Type'] = 'text/html; charset=utf-8'
    self.response.write(template.render(path, full_env))

  def authentication_error(self, error):
    """Shows 'Access denied' page."""
    env = {
      'page_title': 'Access Denied',
      'error': error,
    }
    self.reply('auth/admin/access_denied.html', env=env, status=401)

  def authorization_error(self, error):
    """Redirects to login or shows 'Access Denied' page."""
    # Not authenticated or used IP whitelist for auth -> redirect to login.
    # Bots doesn't use UI, and users should always use real accounts.
    ident = api.get_current_identity()
    if ident.is_anonymous or ident.is_bot:
      self.redirect(self.create_login_url(self.request.url))
      return

    # Admin group is empty -> redirect to bootstrap procedure to create it.
    if model.is_empty_group(model.ADMIN_GROUP):
      self.redirect_to('bootstrap')
      return

    # No access.
    env = {
      'page_title': 'Access Denied',
      'error': error,
    }
    self.reply('auth/admin/access_denied.html', env=env, status=403)


class BootstrapHandler(AdminPageHandler):
  """Creates Administrators group (if necessary) and adds current caller to it.

  Requires Appengine level Admin access for its handlers, since Administrators
  group may not exist yet.

  Used during bootstrap of a new service instance.
  """

  @forbid_ui_on_replica
  @api.require(api.is_superuser)
  def get(self):
    env = {
      'page_title': 'Bootstrap',
      'admin_group': model.ADMIN_GROUP,
      'return_url': self.request.get('r') or '',
    }
    self.reply('auth/admin/bootstrap.html', env)

  @forbid_ui_on_replica
  @api.require(api.is_superuser)
  def post(self):
    added = model.bootstrap_group(
        model.ADMIN_GROUP, [api.get_current_identity()],
        'Users that can manage groups')
    env = {
      'page_title': 'Bootstrap',
      'admin_group': model.ADMIN_GROUP,
      'added': added,
      'return_url': self.request.get('return_url') or '',
    }
    self.reply('auth/admin/bootstrap_done.html', env)


class BootstrapOAuthHandler(AdminPageHandler):
  """Page to set OAuth2 client ID used by the main web UI.

  Requires Appengine level Admin access for its handlers, since without client
  ID there's no UI yet to configure Administrators group.

  Used during bootstrap of a new service instance. Unlike /auth/bootstrap, it is
  also available after the service is linked to some primary Auth service.
  """

  @api.require(api.is_superuser)
  def get(self):
    self.show_page(web_client_id=api.get_web_client_id_uncached())

  @api.require(api.is_superuser)
  def post(self):
    web_client_id = self.request.POST['web_client_id']
    api.set_web_client_id(web_client_id)
    self.show_page(web_client_id=web_client_id, saved=True)

  def show_page(self, web_client_id, saved=False):
    env = {
      'page_title': 'OAuth2 web client ID',
      'web_client_id': web_client_id or '',
      'saved': saved,
    }
    self.reply('auth/admin/bootstrap_oauth.html', env)


class LinkToPrimaryHandler(AdminPageHandler):
  """A page with confirmation of Primary <-> Replica linking request.

  URL to that page is generated by a Primary service.
  """

  def decode_link_ticket(self):
    """Extracts ServiceLinkTicket from 't' GET parameter."""
    try:
      return replication.decode_link_ticket(
          self.request.get('t').encode('ascii'))
    except (KeyError, ValueError):
      self.abort(400)
      return

  @forbid_ui_on_replica
  @api.require(api.is_superuser)
  def get(self):
    ticket = self.decode_link_ticket()
    env = {
      'generated_by': ticket.generated_by,
      'page_title': 'Switch',
      'primary_id': ticket.primary_id,
      'primary_url': ticket.primary_url,
    }
    self.reply('auth/admin/linking.html', env)

  @forbid_ui_on_replica
  @api.require(api.is_superuser)
  def post(self):
    ticket = self.decode_link_ticket()
    success = True
    error_msg = None
    try:
      replication.become_replica(ticket, api.get_current_identity())
    except replication.ProtocolError as exc:
      success = False
      error_msg = exc.message
    env = {
      'error_msg': error_msg,
      'page_title': 'Switch',
      'primary_id': ticket.primary_id,
      'primary_url': ticket.primary_url,
      'success': success,
    }
    self.reply('auth/admin/linking_done.html', env)


################################################################################
## Web UI routes.

# TODO(vadimsh): Switch them to use OAuth for authentication.


class UIHandler(handler.AuthenticatingHandler):
  """Renders Jinja templates extending base.html."""

  # TODO(vadimsh): Enable CSP nonce for styles too. We'll need to get rid of
  # all 'style=...' attributes first.
  csp_use_script_nonce = True

  def reply(self, path, env=None, status=200):
    """Renders template |path| to the HTTP response using given environment.

    Optional keys from |env| that base.html uses:
      css_file: URL to a file with page specific styles, relative to site root.
      js_file: URL to a file with page specific Javascript code, relative to
          site root. File should define global object named same as a filename,
          i.e. '/auth/static/js/api.js' should define global object 'api' that
          incapsulates functionality implemented in the module.
      navbar_tab_id: id of a navbar tab to highlight.
      page_title: title of an HTML page.

    Args:
      path: path to a template, relative to templates/.
      env: additional environment dict to use when rendering the template.
      status: HTTP status code to return.
    """
    env = (env or {}).copy()
    env.setdefault('css_file', None)
    env.setdefault('js_file', None)
    env.setdefault('navbar_tab_id', None)
    env.setdefault('page_title', 'Untitled')

    # This goes to both Jinja2 env and Javascript config object.
    user = self.get_current_user()
    common = {
      'account_picture': user.picture() if user else None,
      'auth_service_config_locked': False, # overridden in auth_service
      'is_admin': api.is_admin(),
      'login_url': self.create_login_url(self.request.url),
      'logout_url': self.create_logout_url('/'),
      'using_gae_auth': self.auth_method == handler.gae_cookie_authentication,
      'xsrf_token': self.generate_xsrf_token(),
    }
    if _ui_data_callback:
      common.update(_ui_data_callback())

    # Name of Javascript module with page code.
    js_module_name = None
    if env['js_file']:
      assert env['js_file'].endswith('.js')
      js_module_name = os.path.basename(env['js_file'])[:-3]

    # This will be accessible from Javascript as global 'config' variable.
    js_config = {
      'identity': api.get_current_identity().to_bytes(),
    }
    js_config.update(common)

    # Jinja2 environment to use to render a template.
    full_env = {
      'app_name': _ui_app_name,
      'app_revision_url': utils.get_app_revision_url(),
      'app_version': utils.get_app_version(),
      'config': json.dumps(js_config),
      'csp_nonce': self.csp_nonce,
      'identity': api.get_current_identity(),
      'js_module_name': js_module_name,
      'navbar': [
        (cls.navbar_tab_id, cls.navbar_tab_title, cls.navbar_tab_url)
        for cls in _ui_navbar_tabs
        if cls.is_visible()
      ],
    }
    full_env.update(common)
    full_env.update(env)

    # Render it.
    self.response.set_status(status)
    self.response.headers['Content-Type'] = 'text/html; charset=utf-8'
    self.response.write(template.render(path, full_env))

  def authentication_error(self, error):
    """Shows 'Access denied' page."""
    # TODO(vadimsh): This will be deleted once we use Google Sign-In.
    env = {
      'page_title': 'Access Denied',
      'error': error,
    }
    self.reply('auth/access_denied.html', env=env, status=401)

  def authorization_error(self, error):
    """Redirects to login or shows 'Access Denied' page."""
    # TODO(vadimsh): This will be deleted once we use Google Sign-In.
    # Not authenticated or used IP whitelist for auth -> redirect to login.
    # Bots doesn't use UI, and users should always use real accounts.
    ident = api.get_current_identity()
    if ident.is_anonymous or ident.is_bot:
      self.redirect(self.create_login_url(self.request.url))
      return

    # Admin group is empty -> redirect to bootstrap procedure to create it.
    if model.is_empty_group(model.ADMIN_GROUP):
      self.redirect_to('bootstrap')
      return

    # No access.
    env = {
      'page_title': 'Access Denied',
      'error': error,
    }
    self.reply('auth/access_denied.html', env=env, status=403)


class MainHandler(UIHandler):
  """Redirects to first navbar tab."""
  @redirect_ui_on_replica
  @api.require(acl.has_access)
  def get(self):
    assert _ui_navbar_tabs
    self.redirect(_ui_navbar_tabs[0].navbar_tab_url)


class UINavbarTabHandler(UIHandler):
  """Handler for a navbar tab page."""
  # List of routes to register, default is [navbar_tab_url].
  routes = []
  # URL to the tab (relative to site root).
  navbar_tab_url = None
  # ID of the tab, will be used in DOM.
  navbar_tab_id = None
  # Title of the tab, will be used in tab title and page title.
  navbar_tab_title = None
  # Relative URL to CSS file with tab's styles.
  css_file = None
  # Relative URL to javascript file with tab's logic.
  js_file_url = None
  # Path to a Jinja2 template with tab's markup.
  template_file = None

  @redirect_ui_on_replica
  @api.require(acl.has_access)
  def get(self, **_params):
    """Renders page HTML to HTTP response stream."""
    env = {
      'css_file': self.css_file,
      'js_file': self.js_file_url,
      'navbar_tab_id': self.navbar_tab_id,
      'page_title': self.navbar_tab_title,
    }
    self.reply(self.template_file, env)

  @classmethod
  def get_webapp2_routes(cls):
    routes = cls.routes or [cls.navbar_tab_url]
    return [webapp2.Route(r, cls) for r in routes]

  @classmethod
  def is_visible(cls):
    """Subclasses may return False to hide the tab from tab bar."""
    return True


################################################################################
## Default tabs (in order of their appearance in the navbar).


class GroupsHandler(UINavbarTabHandler):
  """Page with Groups management."""
  routes = [
    '/auth/groups',
    '/auth/groups/<group:.*>',  # 'group' is handled by js code
  ]
  navbar_tab_url = '/auth/groups'
  navbar_tab_id = 'groups'
  navbar_tab_title = 'Groups'
  css_file = '/auth/static/css/groups.css'
  js_file_url = '/auth/static/js/groups.js'
  template_file = 'auth/groups.html'


class GroupListingHandler(UINavbarTabHandler):
  """Page with full listing of some single group."""
  routes = ['/auth/listing']
  navbar_tab_url = '/auth/groups'
  navbar_tab_id = 'groups'  # keep 'Groups' tab highlighted
  navbar_tab_title = 'Groups'
  js_file_url = '/auth/static/js/listing.js'
  template_file = 'auth/listing.html'


class ChangeLogHandler(UINavbarTabHandler):
  """Page with a log of changes to some groups."""
  navbar_tab_url = '/auth/change_log'
  navbar_tab_id = 'change_log'
  navbar_tab_title = 'Changes'
  js_file_url = '/auth/static/js/change_log.js'
  template_file = 'auth/change_log.html'

  @classmethod
  def is_visible(cls):
    # Hide 'Change Log' tab if there are no change log indexes in the datastore.
    # It happens on services that use components.auth, but do not modify
    # index.yaml. Don't try too hard to hide the log though. If user happes to
    # stumble on Change log page (e.g. by using direct URL), it handles
    # NeedIndexError gracefully (explaining how to configure indexes).
    return change_log.is_changle_log_indexed()


class LookupHandler(UINavbarTabHandler):
  """Page with UI to lookup groups a principal belongs to."""
  navbar_tab_url = '/auth/lookup'
  navbar_tab_id = 'lookup'
  navbar_tab_title = 'Lookup'
  js_file_url = '/auth/static/js/lookup.js'
  template_file = 'auth/lookup.html'


class OAuthConfigHandler(UINavbarTabHandler):
  """Page with OAuth configuration."""
  navbar_tab_url = '/auth/oauth_config'
  navbar_tab_id = 'oauth_config'
  navbar_tab_title = 'OAuth'
  js_file_url = '/auth/static/js/oauth_config.js'
  template_file = 'auth/oauth_config.html'


class IPWhitelistsHandler(UINavbarTabHandler):
  """Page with IP whitelists configuration."""
  navbar_tab_url = '/auth/ip_whitelists'
  navbar_tab_id = 'ip_whitelists'
  navbar_tab_title = 'IP Whitelists'
  js_file_url = '/auth/static/js/ip_whitelists.js'
  template_file = 'auth/ip_whitelists.html'


class ApiDocHandler(UINavbarTabHandler):
  """Page with API documentation extracted from rest_api.py."""
  navbar_tab_url = '/auth/api'
  navbar_tab_id = 'api'
  navbar_tab_title = 'API'

  # These can be used as 'request_type' and 'response_type' in api_doc.
  doc_types = [
    {
      'name': 'Status',
      'doc': 'Outcome of some operation.',
      'example': {'ok': True},
    },
    {
      'name': 'Self info',
      'doc': 'Information about the requester.',
      'example': {
        'identity': 'user:someone@example.com',
        'ip': '192.168.0.1',
      },
    },
    {
      'name': 'Group',
      'doc': 'Represents a group, as stored in the database.',
      'example': {
        'group': {
          'caller_can_modify': True,
          'created_by': 'user:someone@example.com',
          'created_ts': 1409250754978540,
          'description': 'Some free form description',
          'globs': ['user:*@example.com'],
          'members': ['user:a@example.com', 'anonymous:anonymous'],
          'modified_by': 'user:someone@example.com',
          'modified_ts': 1470871200558130,
          'name': 'Some group',
          'nested': ['Some nested group', 'Another nested group'],
          'owners': 'Owning group',
        },
      },
    },
    {
      'name': 'Groups',
      'doc':
        'All groups, along with their metadata. Does not include members '
        'listings.',
      'example': {
        'groups': [
          {
            'caller_can_modify': True,
            'created_by': 'user:someone@example.com',
            'created_ts': 1409250754978540,
            'description': 'Some free form description',
            'modified_by': 'user:someone@example.com',
            'modified_ts': 1470871200558130,
            'name': 'Some group',
            'owners': 'Owning group',
          },
          {
            'caller_can_modify': True,
            'created_by': 'user:someone@example.com',
            'created_ts': 1409250754978540,
            'description': 'Another description',
            'modified_by': 'user:someone@example.com',
            'modified_ts': 1470871200558130,
            'name': 'Another group',
            'owners': 'Owning group',
          },
        ],
      },
    },
    {
      'name': 'Group listing',
      'doc':
        'Recursive listing of all members, globs and nested groups inside '
        'a group. In no particular order.',
      'example': {
        'listing': {
          'members': [
            {'principal': 'user:someone@example.com'},
            {'principal': 'user:another@example.com'},
          ],
          'globs': [
            {'principal': 'user:*@example.com'},
          ],
          'nested': [
            {'principal': 'Nested group'},
            {'principal': 'Another nested group'},
          ],
        },
      },
    },
    {
      'name': 'Group subgraph',
      'doc':
        'Subgraph with all groups that include a principal (perhaps indirectly)'
        ' or owned by it (also perhaps indirectly). Each node has an ID that '
        'matches its index in "nodes" array. These IDs are referenced in '
        '"edges" relations. ID of the node that matches the principal of '
        'interest is 0, i.e it is always first in "nodes" array.',
      'example': {
        'subgraph': {
          'nodes': [
            {
              'kind': 'IDENTITY',
              'edges': {
                'IN': [1, 2],
              },
              'value': 'user:someone@example.com',
            },
            {
              'kind': 'GLOB',
              'edges': {
                'IN': [2],
              },
              'value': 'user:*',
            },
            {
              'kind': 'GROUP',
              'edges': {
                'IN': [3],
                'OWNS': [2, 4],
              },
              'value': 'owners-group',
            },
            {
              'kind': 'GROUP',
              'edges': {
                'OWNS': [3],
              },
              'value': 'another-owners-group',
            },
            {
              'kind': 'GROUP',
              'value': 'owned-group',
            },
          ],
        },
      },
    },
  ]

  @redirect_ui_on_replica
  @api.require(acl.has_access)
  def get(self):
    """Extracts API doc for registered webapp2 API routes."""
    doc_types = []

    def add_doc_type(tp):
      """Adds a request or response format definition to the documentation page.

      'tp' can either reference a globally known doc type by name
      (see ApiDocHandler.doc_types), or can itself be a dict with doc type
      definition.

      Returns the name of the doc type.
      """
      if not tp:
        return None
      # If referenced by name, try to find it among globally known types.
      if isinstance(tp, basestring):
        for d in self.doc_types:
          if d['name'] == tp:
            tp = d
            break
        else:
          return tp  # not found, return original name as is
      # Add, if not already there. Serialize the example first, since doing it
      # from Jinja is a bit more complicated.
      if not any(d['name'] == tp['name'] for d in doc_types):
        tp = tp.copy()
        tp['example'] = json.dumps(
            tp['example'], sort_keys=True, separators=(', ', ': '), indent=2)
        doc_types.append(tp)
      return tp['name']

    api_methods = []
    for route in rest_api.get_rest_api_routes():
      # Remove API parameter regexps from route template, they are mostly noise.
      simplified = re.sub(r'\:.*\>', '>', route.template)
      for doc in getattr(route.handler, 'api_doc', []):
        path = simplified
        if 'params' in doc:
          path += '?' + doc['params']
        api_methods.append({
          'verb': doc['verb'],
          'path': path,
          'doc': doc['doc'],
          'request_type': add_doc_type(doc.get('request_type')),
          'response_type': add_doc_type(doc.get('response_type')),
        })

    env = {
      'navbar_tab_id': self.navbar_tab_id,
      'page_title': self.navbar_tab_title,
      'api_methods': api_methods,
      'doc_types': doc_types,
    }
    self.reply('auth/api.html', env)


# Register them as default tabs. Order is important.
_ui_navbar_tabs = (
  GroupsHandler,
  ChangeLogHandler,
  LookupHandler,
  OAuthConfigHandler,
  IPWhitelistsHandler,
  ApiDocHandler,
)
