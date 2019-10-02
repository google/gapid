# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""GCE Backend cron jobs."""

import logging

import webapp2

from components import decorators

import catalog
import cleanup
import config
import disks
import instance_group_managers
import instance_templates
import instances
import parse
import snapshots


class CatalogedInstanceRemovalHandler(webapp2.RequestHandler):
  """Worker for removing cataloged instances."""

  @decorators.require_cronjob
  def get(self):
    catalog.schedule_removal()


class CatalogedInstanceUpdateHandler(webapp2.RequestHandler):
  """Worker for updating information about cataloged instances."""

  @decorators.require_cronjob
  def get(self):
    catalog.schedule_cataloged_instance_update()


class ConfigImportHandler(webapp2.RequestHandler):
  """Worker for importing the config."""

  @decorators.require_cronjob
  def get(self):
    config.update_template_configs()


class ConfigProcessHandler(webapp2.RequestHandler):
  """Worker for processing the config."""

  @decorators.require_cronjob
  def get(self):
    template_config, manager_config = config.Configuration.load()
    parse.parse(
        template_config.templates,
        manager_config.managers,
        max_concurrent=10,
        max_concurrent_igm=10,
    )


class DiskCreationHandler(webapp2.RequestHandler):
  """Worker for creating disks."""

  @decorators.require_cronjob
  def get(self):
    disks.schedule_creation()


class EntityCleanupHandler(webapp2.RequestHandler):
  """Worker for cleaning up datastore entities."""

  @decorators.require_cronjob
  def get(self):
    cleanup.schedule_deleted_instance_check()
    cleanup.schedule_deleted_instance_cleanup()
    cleanup.schedule_drained_instance_cleanup()
    cleanup.cleanup_instance_group_managers()
    cleanup.cleanup_instance_template_revisions()
    cleanup.cleanup_instance_templates()


class InstanceCatalogHandler(webapp2.RequestHandler):
  """Worker for cataloging instances."""

  @decorators.require_cronjob
  def get(self):
    catalog.schedule_catalog()


class InstanceDeletionHandler(webapp2.RequestHandler):
  """Worker for deleting instances."""

  @decorators.require_cronjob
  def get(self):
    instances.schedule_deletion()


class InstanceFetchHandler(webapp2.RequestHandler):
  """Worker for fetching instances."""

  @decorators.require_cronjob
  def get(self):
    instances.schedule_fetch()


class InstanceGroupManagerCreationHandler(webapp2.RequestHandler):
  """Worker for creating instance group managers."""

  @decorators.require_cronjob
  def get(self):
    instance_group_managers.schedule_creation()


class InstanceGroupManagerDeletionHandler(webapp2.RequestHandler):
  """Worker for deleting instance group managers."""

  @decorators.require_cronjob
  def get(self):
    instance_group_managers.schedule_deletion()


class InstanceGroupResizeHandler(webapp2.RequestHandler):
  """Worker for resizing managed instance groups."""

  @decorators.require_cronjob
  def get(self):
    instance_group_managers.schedule_resize()


class InstancesPendingDeletionDeletionHandler(webapp2.RequestHandler):
  """Worker for deleting instances pending deletion."""

  @decorators.require_cronjob
  def get(self):
    instances.schedule_pending_deletion()


class InstanceTemplateCreationHandler(webapp2.RequestHandler):
  """Worker for creating instance templates."""

  @decorators.require_cronjob
  def get(self):
    instance_templates.schedule_creation()


class InstanceTemplateDeletionHandler(webapp2.RequestHandler):
  """Worker for deleting instance templates."""

  @decorators.require_cronjob
  def get(self):
    instance_templates.schedule_deletion()


class SnapshotFetchHandler(webapp2.RequestHandler):
  """Worker for fetching snapshots."""

  @decorators.require_cronjob
  def get(self):
    snapshots.schedule_fetch()


def create_cron_app():
  return webapp2.WSGIApplication([
      ('/internal/cron/catalog-instances', InstanceCatalogHandler),
      ('/internal/cron/cleanup-entities', EntityCleanupHandler),
      ('/internal/cron/create-disks', DiskCreationHandler),
      ('/internal/cron/create-instance-group-managers',
       InstanceGroupManagerCreationHandler),
      ('/internal/cron/create-instance-templates',
       InstanceTemplateCreationHandler),
      ('/internal/cron/delete-instance-group-managers',
       InstanceGroupManagerDeletionHandler),
      ('/internal/cron/delete-instances-pending-deletion',
       InstancesPendingDeletionDeletionHandler),
      ('/internal/cron/delete-instance-templates',
       InstanceTemplateDeletionHandler),
      ('/internal/cron/delete-instances', InstanceDeletionHandler),
      ('/internal/cron/fetch-instances', InstanceFetchHandler),
      ('/internal/cron/fetch-snapshots', SnapshotFetchHandler),
      ('/internal/cron/import-config', ConfigImportHandler),
      ('/internal/cron/process-config', ConfigProcessHandler),
      ('/internal/cron/remove-cataloged-instances',
       CatalogedInstanceRemovalHandler),
      ('/internal/cron/resize-instance-groups', InstanceGroupResizeHandler),
      ('/internal/cron/update-cataloged-instances',
       CatalogedInstanceUpdateHandler),
  ])
