# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Generates the swarming_bot.zip archive for the bot.

Unlike the other source files, this file can be run from ../tools/bot_archive.py
stand-alone to generate a swarming_bot.zip for local testing so it doesn't
import anything from the AppEngine SDK.

The hash of the content of the files in the archive is used to define the
current version of the swarming bot code.
"""

import hashlib
import json
import logging
import os
import StringIO
import zipfile


# List of files needed by the swarming bot.
# TODO(maruel): Make the list automatically generated?
FILES = (
    '__main__.py',
    'adb/__init__.py',
    'adb/adb_commands.py',
    'adb/adb_protocol.py',
    'adb/common.py',
    'adb/contrib/__init__.py',
    'adb/contrib/adb_commands_safe.py',
    'adb/contrib/high.py',
    'adb/contrib/parallel.py',
    'adb/fastboot.py',
    'adb/filesync_protocol.py',
    'adb/sign_pythonrsa.py',
    'adb/usb_exceptions.py',
    'api/__init__.py',
    'api/bot.py',
    'api/oauth.py',
    'api/os_utilities.py',
    'api/parallel.py',
    'api/platforms/__init__.py',
    'api/platforms/android.py',
    'api/platforms/common.py',
    'api/platforms/gce.py',
    'api/platforms/gpu.py',
    'api/platforms/linux.py',
    'api/platforms/osx.py',
    'api/platforms/posix.py',
    'api/platforms/win.py',
    'bot_code/__init__.py',
    'bot_code/bot_auth.py',
    'bot_code/bot_main.py',
    'bot_code/common.py',
    'bot_code/file_reader.py',
    'bot_code/file_refresher.py',
    'bot_code/remote_client.py',
    'bot_code/remote_client_errors.py',
    'bot_code/remote_client_grpc.py',
    'bot_code/singleton.py',
    'bot_code/task_runner.py',
    'client/auth.py',
    'client/cipd.py',
    'client/isolate_storage.py',
    'client/isolated_format.py',
    'client/isolateserver.py',
    'client/local_caching.py',
    'client/proto/__init__.py',
    'client/proto/bytestream_pb2.py',
    'client/run_isolated.py',
    'config/__init__.py',
    'infra_libs/__init__.py',
    'infra_libs/_command_line_linux.py',
    'infra_libs/_command_line_stub.py',
    'infra_libs/httplib2_utils.py',
    'infra_libs/ts_mon/__init__.py',
    'infra_libs/ts_mon/common/__init__.py',
    'infra_libs/ts_mon/common/distribution.py',
    'infra_libs/ts_mon/common/errors.py',
    'infra_libs/ts_mon/common/helpers.py',
    'infra_libs/ts_mon/common/http_metrics.py',
    'infra_libs/ts_mon/common/interface.py',
    'infra_libs/ts_mon/common/metric_store.py',
    'infra_libs/ts_mon/common/metrics.py',
    'infra_libs/ts_mon/common/monitors.py',
    'infra_libs/ts_mon/common/pb_to_popo.py',
    'infra_libs/ts_mon/common/standard_metrics.py',
    'infra_libs/ts_mon/common/targets.py',
    'infra_libs/ts_mon/config.py',
    'infra_libs/ts_mon/protos/__init__.py',
    'infra_libs/ts_mon/protos/acquisition_network_device_pb2.py',
    'infra_libs/ts_mon/protos/acquisition_task_pb2.py',
    'infra_libs/ts_mon/protos/any_pb2.py',
    'infra_libs/ts_mon/protos/metrics_pb2.py',
    'infra_libs/ts_mon/protos/timestamp_pb2.py',
    'infra_libs/utils.py',
    'libs/__init__.py',
    'libs/luci_context/__init__.py',
    'libs/luci_context/luci_context.py',
    'proto_bot/__init__.py',
    'proto_bot/bots_pb2.py',
    'proto_bot/bots_pb2_grpc.py',
    'proto_bot/bytestream_pb2.py',
    'proto_bot/bytestream_pb2_grpc.py',
    'proto_bot/code_pb2.py',
    'proto_bot/command_pb2.py',
    'proto_bot/status_pb2.py',
    'proto_bot/tasks_pb2.py',
    'proto_bot/tasks_pb2_grpc.py',
    'python_libusb1/__init__.py',
    'python_libusb1/libusb1.py',
    'python_libusb1/usb1.py',
    'signal_trace.py',
    'third_party/__init__.py',
    'third_party/cachetools/abc.py',
    'third_party/cachetools/cache.py',
    'third_party/cachetools/func.py',
    'third_party/cachetools/__init__.py',
    'third_party/cachetools/keys.py',
    'third_party/cachetools/lfu.py',
    'third_party/cachetools/lru.py',
    'third_party/cachetools/rr.py',
    'third_party/cachetools/ttl.py',
    'third_party/colorama/__init__.py',
    'third_party/colorama/ansi.py',
    'third_party/colorama/ansitowin32.py',
    'third_party/colorama/initialise.py',
    'third_party/colorama/win32.py',
    'third_party/colorama/winterm.py',
    'third_party/depot_tools/__init__.py',
    'third_party/depot_tools/fix_encoding.py',
    'third_party/depot_tools/subcommand.py',
    'third_party/google/__init__.py',
    'third_party/google/auth/app_engine.py',
    'third_party/google/auth/_cloud_sdk.py',
    'third_party/google/auth/credentials.py',
    'third_party/google/auth/crypt.py',
    'third_party/google/auth/_default.py',
    'third_party/google/auth/environment_vars.py',
    'third_party/google/auth/exceptions.py',
    'third_party/google/auth/_helpers.py',
    'third_party/google/auth/iam.py',
    'third_party/google/auth/__init__.py',
    'third_party/google/auth/jwt.py',
    'third_party/google/auth/_oauth2client.py',
    'third_party/google/auth/_service_account_info.py',
    'third_party/google/auth/compute_engine/credentials.py',
    'third_party/google/auth/compute_engine/__init__.py',
    'third_party/google/auth/compute_engine/_metadata.py',
    'third_party/google/auth/transport/grpc.py',
    'third_party/google/auth/transport/_http_client.py',
    'third_party/google/auth/transport/__init__.py',
    'third_party/google/auth/transport/requests.py',
    'third_party/google/auth/transport/urllib3.py',
    'third_party/google/oauth2/_client.py',
    'third_party/google/oauth2/credentials.py',
    'third_party/google/oauth2/id_token.py',
    'third_party/google/oauth2/__init__.py',
    'third_party/google/oauth2/service_account.py',
    'third_party/google/protobuf/__init__.py',
    'third_party/google/protobuf/any_pb2.py',
    'third_party/google/protobuf/descriptor.py',
    'third_party/google/protobuf/descriptor_database.py',
    'third_party/google/protobuf/descriptor_pb2.py',
    'third_party/google/protobuf/descriptor_pool.py',
    'third_party/google/protobuf/duration_pb2.py',
    'third_party/google/protobuf/empty_pb2.py',
    'third_party/google/protobuf/field_mask_pb2.py',
    'third_party/google/protobuf/internal/__init__.py',
    'third_party/google/protobuf/internal/api_implementation.py',
    'third_party/google/protobuf/internal/containers.py',
    'third_party/google/protobuf/internal/decoder.py',
    'third_party/google/protobuf/internal/encoder.py',
    'third_party/google/protobuf/internal/enum_type_wrapper.py',
    'third_party/google/protobuf/internal/message_listener.py',
    'third_party/google/protobuf/internal/python_message.py',
    'third_party/google/protobuf/internal/type_checkers.py',
    'third_party/google/protobuf/internal/well_known_types.py',
    'third_party/google/protobuf/internal/wire_format.py',
    'third_party/google/protobuf/json_format.py',
    'third_party/google/protobuf/message.py',
    'third_party/google/protobuf/message_factory.py',
    'third_party/google/protobuf/reflection.py',
    'third_party/google/protobuf/struct_pb2.py',
    'third_party/google/protobuf/symbol_database.py',
    'third_party/google/protobuf/text_encoding.py',
    'third_party/google/protobuf/text_format.py',
    'third_party/google/protobuf/timestamp_pb2.py',
    'third_party/googleapiclient/__init__.py',
    'third_party/googleapiclient/channel.py',
    'third_party/googleapiclient/discovery.py',
    'third_party/googleapiclient/discovery_cache/__init__.py',
    'third_party/googleapiclient/discovery_cache/appengine_memcache.py',
    'third_party/googleapiclient/discovery_cache/base.py',
    'third_party/googleapiclient/discovery_cache/file_cache.py',
    'third_party/googleapiclient/errors.py',
    'third_party/googleapiclient/http.py',
    'third_party/googleapiclient/mimeparse.py',
    'third_party/googleapiclient/model.py',
    'third_party/googleapiclient/sample_tools.py',
    'third_party/googleapiclient/schema.py',
    'third_party/httplib2/__init__.py',
    'third_party/httplib2/cacerts.txt',
    'third_party/httplib2/certs.py',
    'third_party/httplib2/iri2uri.py',
    'third_party/httplib2/socks.py',
    'third_party/oauth2client/__init__.py',
    'third_party/oauth2client/_helpers.py',
    'third_party/oauth2client/_openssl_crypt.py',
    'third_party/oauth2client/_pycrypto_crypt.py',
    'third_party/oauth2client/client.py',
    'third_party/oauth2client/clientsecrets.py',
    'third_party/oauth2client/crypt.py',
    'third_party/oauth2client/file.py',
    'third_party/oauth2client/gce.py',
    'third_party/oauth2client/keyring_storage.py',
    'third_party/oauth2client/locked_file.py',
    'third_party/oauth2client/multistore_file.py',
    'third_party/oauth2client/service_account.py',
    'third_party/oauth2client/tools.py',
    'third_party/oauth2client/util.py',
    'third_party/oauth2client/xsrfutil.py',
    'third_party/pyasn1-modules/pyasn1_modules/__init__.py',
    'third_party/pyasn1-modules/pyasn1_modules/pem.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc1155.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc1157.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc1901.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc1902.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc1905.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc2251.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc2314.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc2315.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc2437.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc2459.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc2511.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc2560.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc2986.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc3279.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc3280.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc3281.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc3412.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc3414.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc3447.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc3852.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc4210.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc4211.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc5083.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc5084.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc5208.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc5280.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc5652.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc6402.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc8103.py',
    'third_party/pyasn1-modules/pyasn1_modules/rfc8226.py',
    'third_party/pyasn1/pyasn1/__init__.py',
    'third_party/pyasn1/pyasn1/codec/__init__.py',
    'third_party/pyasn1/pyasn1/codec/ber/__init__.py',
    'third_party/pyasn1/pyasn1/codec/ber/decoder.py',
    'third_party/pyasn1/pyasn1/codec/ber/encoder.py',
    'third_party/pyasn1/pyasn1/codec/ber/eoo.py',
    'third_party/pyasn1/pyasn1/codec/cer/__init__.py',
    'third_party/pyasn1/pyasn1/codec/cer/decoder.py',
    'third_party/pyasn1/pyasn1/codec/cer/encoder.py',
    'third_party/pyasn1/pyasn1/codec/der/__init__.py',
    'third_party/pyasn1/pyasn1/codec/der/decoder.py',
    'third_party/pyasn1/pyasn1/codec/der/encoder.py',
    'third_party/pyasn1/pyasn1/codec/native/__init__.py',
    'third_party/pyasn1/pyasn1/codec/native/decoder.py',
    'third_party/pyasn1/pyasn1/codec/native/encoder.py',
    'third_party/pyasn1/pyasn1/compat/__init__.py',
    'third_party/pyasn1/pyasn1/compat/binary.py',
    'third_party/pyasn1/pyasn1/compat/calling.py',
    'third_party/pyasn1/pyasn1/compat/dateandtime.py',
    'third_party/pyasn1/pyasn1/compat/integer.py',
    'third_party/pyasn1/pyasn1/compat/octets.py',
    'third_party/pyasn1/pyasn1/compat/string.py',
    'third_party/pyasn1/pyasn1/debug.py',
    'third_party/pyasn1/pyasn1/error.py',
    'third_party/pyasn1/pyasn1/type/__init__.py',
    'third_party/pyasn1/pyasn1/type/base.py',
    'third_party/pyasn1/pyasn1/type/char.py',
    'third_party/pyasn1/pyasn1/type/constraint.py',
    'third_party/pyasn1/pyasn1/type/error.py',
    'third_party/pyasn1/pyasn1/type/namedtype.py',
    'third_party/pyasn1/pyasn1/type/namedval.py',
    'third_party/pyasn1/pyasn1/type/opentype.py',
    'third_party/pyasn1/pyasn1/type/tag.py',
    'third_party/pyasn1/pyasn1/type/tagmap.py',
    'third_party/pyasn1/pyasn1/type/univ.py',
    'third_party/pyasn1/pyasn1/type/useful.py',
    'third_party/requests/__init__.py',
    'third_party/requests/adapters.py',
    'third_party/requests/api.py',
    'third_party/requests/auth.py',
    'third_party/requests/certs.py',
    'third_party/requests/compat.py',
    'third_party/requests/cookies.py',
    'third_party/requests/exceptions.py',
    'third_party/requests/hooks.py',
    'third_party/requests/models.py',
    'third_party/requests/packages/__init__.py',
    'third_party/requests/packages/urllib3/__init__.py',
    'third_party/requests/packages/urllib3/_collections.py',
    'third_party/requests/packages/urllib3/connection.py',
    'third_party/requests/packages/urllib3/connectionpool.py',
    'third_party/requests/packages/urllib3/contrib/__init__.py',
    'third_party/requests/packages/urllib3/contrib/ntlmpool.py',
    'third_party/requests/packages/urllib3/contrib/pyopenssl.py',
    'third_party/requests/packages/urllib3/exceptions.py',
    'third_party/requests/packages/urllib3/fields.py',
    'third_party/requests/packages/urllib3/filepost.py',
    'third_party/requests/packages/urllib3/packages/__init__.py',
    'third_party/requests/packages/urllib3/packages/ordered_dict.py',
    'third_party/requests/packages/urllib3/packages/six.py',
    'third_party/requests/packages/urllib3/packages/ssl_match_hostname/'
        '__init__.py',
    'third_party/requests/packages/urllib3/packages/ssl_match_hostname/'
        '_implementation.py',
    'third_party/requests/packages/urllib3/poolmanager.py',
    'third_party/requests/packages/urllib3/request.py',
    'third_party/requests/packages/urllib3/response.py',
    'third_party/requests/packages/urllib3/util/__init__.py',
    'third_party/requests/packages/urllib3/util/connection.py',
    'third_party/requests/packages/urllib3/util/request.py',
    'third_party/requests/packages/urllib3/util/response.py',
    'third_party/requests/packages/urllib3/util/retry.py',
    'third_party/requests/packages/urllib3/util/ssl_.py',
    'third_party/requests/packages/urllib3/util/timeout.py',
    'third_party/requests/packages/urllib3/util/url.py',
    'third_party/requests/sessions.py',
    'third_party/requests/status_codes.py',
    'third_party/requests/structures.py',
    'third_party/requests/utils.py',
    'third_party/rsa/rsa/__init__.py',
    'third_party/rsa/rsa/_compat.py',
    'third_party/rsa/rsa/_version133.py',
    'third_party/rsa/rsa/_version200.py',
    'third_party/rsa/rsa/asn1.py',
    'third_party/rsa/rsa/bigfile.py',
    'third_party/rsa/rsa/cli.py',
    'third_party/rsa/rsa/common.py',
    'third_party/rsa/rsa/core.py',
    'third_party/rsa/rsa/key.py',
    'third_party/rsa/rsa/parallel.py',
    'third_party/rsa/rsa/pem.py',
    'third_party/rsa/rsa/pkcs1.py',
    'third_party/rsa/rsa/prime.py',
    'third_party/rsa/rsa/randnum.py',
    'third_party/rsa/rsa/transform.py',
    'third_party/rsa/rsa/util.py',
    'third_party/rsa/rsa/varblock.py',
    'third_party/six/__init__.py',
    'third_party/uritemplate/__init__.py',
    'utils/__init__.py',
    'utils/auth_server.py',
    'utils/authenticators.py',
    'utils/cacert.pem',
    'utils/file_path.py',
    'utils/fs.py',
    'utils/grpc_proxy.py',
    'utils/large.py',
    'utils/logging_utils.py',
    'utils/lru.py',
    'utils/net.py',
    'utils/oauth.py',
    'utils/on_error.py',
    'utils/subprocess42.py',
    'utils/threading_utils.py',
    'utils/tools.py',
    'utils/zip_package.py',
)


def is_windows():
  """Returns True if this code is running under Windows."""
  return os.__file__[0] != '/'


def resolve_symlink(path):
  """Processes path containing symlink on Windows.

  This is needed to make ../swarming_bot/main_test.py pass on Windows because
  git on Windows renders symlinks as normal files.
  """
  if not is_windows():
    # Only does this dance on Windows.
    return path
  parts = os.path.normpath(path).split(os.path.sep)
  for i in xrange(2, len(parts)):
    partial = os.path.sep.join(parts[:i])
    if os.path.isfile(partial):
      with open(partial) as f:
        link = f.read()
      assert '\n' not in link and link, link
      parts[i-1] = link
  return os.path.normpath(os.path.sep.join(parts))


def yield_swarming_bot_files(
    root_dir, host, host_version, additionals, settings):
  """Yields all the files to map as tuple(filename, content).

  config.json is injected with json data about the server.

  This function guarantees that the output is sorted by filename.
  """
  items = {i: None for i in FILES}
  items.update(additionals)
  items['config/config.json'] = _make_config_json(host, host_version, settings)
  logging.debug('Bot config.json: %s', items['config/config.json'])
  for item, content in sorted(items.iteritems()):
    if content is not None:
      yield item, content
    else:
      with open(resolve_symlink(os.path.join(root_dir, item)), 'rb') as f:
        yield item, f.read()


def get_swarming_bot_zip(
    root_dir, host, host_version, additionals, settings):
  """Returns a zipped file of all the files a bot needs to run.

  Arguments:
    root_dir: directory swarming_bot.
    additionals: dict(filepath: content) of additional items to put into the zip
        file, in addition to FILES and MAPPED. In practice, it's going to be a
        custom bot_config.py.
    enable_ts_monitoring: bool if ts_mon should be enabled on the bot.

  Returns:
    Tuple(str being the zipped file's content, bot version (SHA256) it
    represents).
  """
  zip_memory_file = StringIO.StringIO()
  h = hashlib.sha256()
  with zipfile.ZipFile(zip_memory_file, 'w', zipfile.ZIP_DEFLATED) as zip_file:
    for name, content in yield_swarming_bot_files(
        root_dir, host, host_version, additionals, settings):

      # We must pass ZipInfo object, otherwise zipfile will generate ZipInfo
      # on its own, setting date_time to the current time, thus making the zip
      # archive non-deterministic. See zipfile.py source for where external_attr
      # comes from. It is copy-pasta from there.
      zinfo = zipfile.ZipInfo(filename=name)
      zinfo.compress_type = zip_file.compression
      if zinfo.filename.endswith('/'):
        zinfo.external_attr = 0o40775 << 16   # drwxrwxr-x
        zinfo.external_attr |= 0x10           # MS-DOS directory flag
      else:
        zinfo.external_attr = 0o600 << 16     # ?rw-------
      zip_file.writestr(zinfo, content)

      h.update(str(len(name)))
      h.update(name)
      h.update(str(len(content)))
      h.update(content)

  data = zip_memory_file.getvalue()
  bot_version = h.hexdigest()
  logging.info(
      'get_swarming_bot_zip(%s) is %d bytes; %s',
      additionals.keys(), len(data), bot_version)
  return data, bot_version


def get_swarming_bot_version(
    root_dir, host, host_version, additionals, settings):
  """Returns the SHA256 hash of the bot code, representing the version.

  Arguments:
    root_dir: directory swarming_bot.
    additionals: See get_swarming_bot_zip's doc.
    enable_ts_monitoring: bool if ts_mon should be enabled on the bot.

  Returns:
    The SHA256 hash of the bot code.
  """
  h = hashlib.sha256()
  try:
    # TODO(maruel): Deduplicate from zip_package.genereate_version().
    for name, content in yield_swarming_bot_files(
        root_dir, host, host_version, additionals, settings):
      h.update(str(len(name)))
      h.update(name)
      h.update(str(len(content)))
      h.update(content)
  except IOError:
    logging.warning('Missing expected file. Hash will be invalid.')
  bot_version = h.hexdigest()
  logging.info(
      'get_swarming_bot_version(%s) = %s', sorted(additionals), bot_version)
  return bot_version


## Private stuff.


def _make_config_json(host, host_version, settings):
  """Generates a config.json to embed in swarming_bot.zip"""
  # The keys must match ../swarming_bot/config/config.json.
  config = {
    'enable_ts_monitoring': False,
    'isolate_grpc_proxy': "",
    'server': host.rstrip('/'),
    'server_version': host_version,
    'swarming_grpc_proxy': "",
  }
  if settings:
    config['enable_ts_monitoring'] = settings.enable_ts_monitoring
    config['isolate_grpc_proxy'] = settings.bot_isolate_grpc_proxy
    config['swarming_grpc_proxy'] = settings.bot_swarming_grpc_proxy
  return json.dumps(config)
