# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Provides functions: get_native_path_case(), isabs() and safe_join().

This module assumes that filesystem is not changing while current process
is running and thus it caches results of functions that depend on FS state.
"""

import ctypes
import getpass
import logging
import os
import posixpath
import re
import shlex
import stat
import sys
import tempfile
import time
import unicodedata

from utils import fs
from utils import subprocess42
from utils import tools


# Types of action accepted by link_file().
HARDLINK, HARDLINK_WITH_FALLBACK, SYMLINK, SYMLINK_WITH_FALLBACK, COPY = range(
    1, 6)


## OS-specific imports


if sys.platform == 'win32':
  import locale
  from ctypes import wintypes  # pylint: disable=ungrouped-imports
  from ctypes.wintypes import windll  # pylint: disable=ungrouped-imports
elif sys.platform == 'darwin':
  import Carbon.File
  import MacOS


if sys.platform == 'win32':
  class LUID(ctypes.Structure):
    _fields_ = [
      ('low_part', wintypes.DWORD), ('high_part', wintypes.LONG),
    ]


  class LUID_AND_ATTRIBUTES(ctypes.Structure):
    _fields_ = [('LUID', LUID), ('attributes', wintypes.DWORD)]


  class TOKEN_PRIVILEGES(ctypes.Structure):
    _fields_ = [
      ('count', wintypes.DWORD), ('privileges', LUID_AND_ATTRIBUTES*0),
    ]

    def get_array(self):
      array_type = LUID_AND_ATTRIBUTES * self.count
      return ctypes.cast(self.privileges, ctypes.POINTER(array_type)).contents


  GetCurrentProcess = windll.kernel32.GetCurrentProcess
  GetCurrentProcess.restype = wintypes.HANDLE
  OpenProcessToken = windll.advapi32.OpenProcessToken
  OpenProcessToken.argtypes = (
      wintypes.HANDLE, wintypes.DWORD, ctypes.POINTER(wintypes.HANDLE))
  OpenProcessToken.restype = wintypes.BOOL
  LookupPrivilegeValue = windll.advapi32.LookupPrivilegeValueW
  LookupPrivilegeValue.argtypes = (
      wintypes.LPCWSTR, wintypes.LPCWSTR, ctypes.POINTER(LUID))
  LookupPrivilegeValue.restype = wintypes.BOOL
  LookupPrivilegeName = windll.advapi32.LookupPrivilegeNameW
  LookupPrivilegeName.argtypes = (
      wintypes.LPCWSTR, ctypes.POINTER(LUID), wintypes.LPWSTR,
      ctypes.POINTER(wintypes.DWORD))
  LookupPrivilegeName.restype = wintypes.BOOL
  PTOKEN_PRIVILEGES = ctypes.POINTER(TOKEN_PRIVILEGES)
  AdjustTokenPrivileges = windll.advapi32.AdjustTokenPrivileges
  AdjustTokenPrivileges.restype = wintypes.BOOL
  AdjustTokenPrivileges.argtypes = (
      wintypes.HANDLE, wintypes.BOOL, PTOKEN_PRIVILEGES,
      wintypes.DWORD, PTOKEN_PRIVILEGES,
      ctypes.POINTER(wintypes.DWORD))


  def FormatError(err):
    """Returns a formatted error on Windows in unicode."""
    # We need to take in account the current code page.
    return ctypes.wintypes.FormatError(err).decode(
        locale.getpreferredencoding(), 'replace')


  def QueryDosDevice(drive_letter):
    """Returns the Windows 'native' path for a DOS drive letter."""
    assert re.match(r'^[a-zA-Z]:$', drive_letter), drive_letter
    assert isinstance(drive_letter, unicode)
    # Guesswork. QueryDosDeviceW never returns the required number of bytes.
    chars = 1024
    drive_letter = drive_letter
    p = wintypes.create_unicode_buffer(chars)
    if 0 == windll.kernel32.QueryDosDeviceW(drive_letter, p, chars):
      err = ctypes.GetLastError()
      if err:
        # pylint: disable=undefined-variable
        msg = u'QueryDosDevice(%s): %s (%d)' % (
              drive_letter, FormatError(err), err)
        raise WindowsError(err, msg.encode('utf-8'))
    return p.value


  def GetShortPathName(long_path):
    """Returns the Windows short path equivalent for a 'long' path."""
    path = fs.extend(long_path)
    chars = windll.kernel32.GetShortPathNameW(path, None, 0)
    if chars:
      p = wintypes.create_unicode_buffer(chars)
      if windll.kernel32.GetShortPathNameW(path, p, chars):
        return fs.trim(p.value)

    err = ctypes.GetLastError()
    if err:
      # pylint: disable=undefined-variable
      msg = u'GetShortPathName(%s): %s (%d)' % (
            long_path, FormatError(err), err)
      raise WindowsError(err, msg.encode('utf-8'))


  def GetLongPathName(short_path):
    """Returns the Windows long path equivalent for a 'short' path."""
    path = fs.extend(short_path)
    chars = windll.kernel32.GetLongPathNameW(path, None, 0)
    if chars:
      p = wintypes.create_unicode_buffer(chars)
      if windll.kernel32.GetLongPathNameW(path, p, chars):
        return fs.trim(p.value)

    err = ctypes.GetLastError()
    if err:
      # pylint: disable=undefined-variable
      msg = u'GetLongPathName(%s): %s (%d)' % (
            short_path, FormatError(err), err)
      raise WindowsError(err, msg.encode('utf-8'))


  def MoveFileEx(oldpath, newpath, flags):
    """Calls MoveFileEx, converting errors to WindowsError exceptions."""
    old_p = fs.extend(oldpath)
    new_p = fs.extend(newpath)
    if not windll.kernel32.MoveFileExW(old_p, new_p, int(flags)):
      # pylint: disable=undefined-variable
      err = ctypes.GetLastError()
      msg = u'MoveFileEx(%s, %s, %d): %s (%d)' % (
            oldpath, newpath, flags, FormatError(err), err)
      raise WindowsError(err, msg.encode('utf-8'))


  class DosDriveMap(object):
    """Maps \Device\HarddiskVolumeN to N: on Windows."""
    # Keep one global cache.
    _MAPPING = {}

    def __init__(self):
      """Lazy loads the cache."""
      if not self._MAPPING:
        # This is related to UNC resolver on windows. Ignore that.
        self._MAPPING[u'\\Device\\Mup'] = None
        self._MAPPING[u'\\SystemRoot'] = os.environ[u'SystemRoot']

        for letter in (chr(l) for l in xrange(ord('C'), ord('Z')+1)):
          try:
            letter = u'%s:' % letter
            mapped = QueryDosDevice(letter)
            if mapped in self._MAPPING:
              logging.warn(
                  ('Two drives: \'%s\' and \'%s\', are mapped to the same disk'
                   '. Drive letters are a user-mode concept and the kernel '
                   'traces only have NT path, so all accesses will be '
                   'associated with the first drive letter, independent of the '
                   'actual letter used by the code') % (
                     self._MAPPING[mapped], letter))
            else:
              self._MAPPING[mapped] = letter
          except WindowsError:  # pylint: disable=undefined-variable
            pass

    def to_win32(self, path):
      """Converts a native NT path to Win32/DOS compatible path."""
      match = re.match(r'(^\\Device\\[a-zA-Z0-9]+)(\\.*)?$', path)
      if not match:
        raise ValueError(
            'Can\'t convert %s into a Win32 compatible path' % path,
            path)
      if not match.group(1) in self._MAPPING:
        # Unmapped partitions may be accessed by windows for the
        # fun of it while the test is running. Discard these.
        return None
      drive = self._MAPPING[match.group(1)]
      if not drive or not match.group(2):
        return drive
      return drive + match.group(2)


  def change_acl_for_delete(path):
    """Zaps the SECURITY_DESCRIPTOR's DACL on a directory entry that is tedious
    to delete.

    This function is a heavy hammer. It discards the SECURITY_DESCRIPTOR and
    creates a new one with only one DACL set to user:FILE_ALL_ACCESS.

    Used as last resort.
    """
    STANDARD_RIGHTS_REQUIRED = 0xf0000
    SYNCHRONIZE = 0x100000
    FILE_ALL_ACCESS = STANDARD_RIGHTS_REQUIRED | SYNCHRONIZE | 0x3ff

    import win32security
    user, _domain, _type = win32security.LookupAccountName(
        '', getpass.getuser())
    sd = win32security.SECURITY_DESCRIPTOR()
    sd.Initialize()
    sd.SetSecurityDescriptorOwner(user, False)
    dacl = win32security.ACL()
    dacl.Initialize()
    dacl.AddAccessAllowedAce(
        win32security.ACL_REVISION_DS, FILE_ALL_ACCESS, user)
    sd.SetSecurityDescriptorDacl(1, dacl, 0)
    # Note that this assumes the object is either owned by the current user or
    # its group or that the current ACL permits this. Otherwise it will silently
    # fail.
    win32security.SetFileSecurity(
        fs.extend(path), win32security.DACL_SECURITY_INFORMATION, sd)
    # It's important to also look for the read only bit after, as it's possible
    # the set_read_only() call to remove the read only bit had silently failed
    # because there was no DACL for the user.
    if not (os.stat(path).st_mode & stat.S_IWUSR):
      os.chmod(path, 0777)


  def isabs(path):
    """Accepts X: as an absolute path, unlike python's os.path.isabs()."""
    return os.path.isabs(path) or len(path) == 2 and path[1] == ':'


  def find_item_native_case(root, item):
    """Gets the native path case of a single item based at root_path."""
    if item == '..':
      return item

    root = get_native_path_case(root)
    return os.path.basename(get_native_path_case(os.path.join(root, item)))


  @tools.profile
  @tools.cached
  def get_native_path_case(p):
    """Returns the native path case for an existing file.

    On Windows, removes any leading '\\?\'.
    """
    assert isinstance(p, unicode), repr(p)
    if not isabs(p):
      raise ValueError(
          'get_native_path_case(%r): Require an absolute path' % p, p)

    # Make sure it is normalized to os.path.sep. Do not do it here to keep the
    # function fast
    assert '/' not in p, p
    suffix = ''
    count = p.count(':')
    if count > 1:
      # This means it has an alternate-data stream. There could be 3 ':', since
      # it could be the $DATA datastream of an ADS. Split the whole ADS suffix
      # off and add it back afterward. There is no way to know the native path
      # case of an alternate data stream.
      items = p.split(':')
      p = ':'.join(items[0:2])
      suffix = ''.join(':' + i for i in items[2:])

    # TODO(maruel): Use os.path.normpath?
    if p.endswith('.\\'):
      p = p[:-2]

    # Windows used to have an option to turn on case sensitivity on non Win32
    # subsystem but that's out of scope here and isn't supported anymore.
    # Go figure why GetShortPathName() is needed.
    try:
      out = GetLongPathName(GetShortPathName(p))
    except OSError, e:
      if e.args[0] in (2, 3, 5):
        # The path does not exist. Try to recurse and reconstruct the path.
        base = os.path.dirname(p)
        rest = os.path.basename(p)
        return os.path.join(get_native_path_case(base), rest)
      raise
    # Always upper case the first letter since GetLongPathName() will return the
    # drive letter in the case it was given.
    return out[0].upper() + out[1:] + suffix


  def get_process_token():
    """Get the current process token."""
    TOKEN_ALL_ACCESS = 0xF01FF
    token = ctypes.wintypes.HANDLE()
    if not OpenProcessToken(
        GetCurrentProcess(), TOKEN_ALL_ACCESS, ctypes.byref(token)):
      # pylint: disable=undefined-variable
      raise WindowsError('Couldn\'t get process token')
    return token


  def get_luid(name):
    """Returns the LUID for a privilege."""
    luid = LUID()
    if not LookupPrivilegeValue(None, unicode(name), ctypes.byref(luid)):
      # pylint: disable=undefined-variable
      raise WindowsError('Couldn\'t lookup privilege value')
    return luid


  def enable_privilege(name):
    """Enables the privilege for the current process token.

    Returns:
    - True if the assignment is successful.
    """
    SE_PRIVILEGE_ENABLED = 2
    ERROR_NOT_ALL_ASSIGNED = 1300

    size = ctypes.sizeof(TOKEN_PRIVILEGES) + ctypes.sizeof(LUID_AND_ATTRIBUTES)
    buf = ctypes.create_string_buffer(size)
    tp = ctypes.cast(buf, ctypes.POINTER(TOKEN_PRIVILEGES)).contents
    tp.count = 1
    tp.get_array()[0].LUID = get_luid(name)
    tp.get_array()[0].Attributes = SE_PRIVILEGE_ENABLED
    token = get_process_token()
    try:
      if not AdjustTokenPrivileges(token, False, tp, 0, None, None):
        # pylint: disable=undefined-variable
        raise WindowsError(
            u'AdjustTokenPrivileges(%r): failed: %s' %
              (name, ctypes.GetLastError()))
    finally:
      windll.kernel32.CloseHandle(token)
    return ctypes.GetLastError() != ERROR_NOT_ALL_ASSIGNED


  def enable_symlink():
    """Enable SeCreateSymbolicLinkPrivilege for the current token.

    This funciton is only helpful in ONE of the following case:
    - UAC is disabled, account is admin OR SeCreateSymbolicLinkPrivilege was
      manually granted.
    - UAC is enabled, account is NOT admin AND SeCreateSymbolicLinkPrivilege was
      manually granted.

    If running Windows 10 and the following is true, then enable_symlink() is
    unnecessary.
    - Windows 10 with build 14971 or later
    - Admin account
    - UAC enabled
    - Developer mode enabled (not the default)

    Returns:
    - True if symlink support is enabled.
    """
    return enable_privilege(u'SeCreateSymbolicLinkPrivilege')


  def kill_children_processes(root):
    """Try to kill all children processes indistriminately and prints updates to
    stderr.

    Returns:
      True if at least one child process was found.
    """
    processes = _get_children_processes_win(root)
    if not processes:
      return False
    sys.stderr.write('Enumerating processes:\n')
    for _, proc in sorted(processes.iteritems()):
      sys.stderr.write(
          '- pid %d; Handles: %d; Exe: %s; Cmd: %s\n' % (
            proc.ProcessId,
            proc.HandleCount,
            proc.ExecutablePath,
            proc.CommandLine))
    sys.stderr.write('Terminating %d processes:\n' % len(processes))
    for pid in sorted(processes):
      try:
        # Killing is asynchronous.
        os.kill(pid, 9)
        sys.stderr.write('- %d killed\n' % pid)
      except OSError as e:
        sys.stderr.write('- failed to kill %s, error %s\n' % (pid, e))
    return True


  ## Windows private code.


  def _enum_processes_win():
    """Returns all processes on the system that are accessible to this process.

    Returns:
      Win32_Process COM objects. See
      http://msdn.microsoft.com/library/aa394372.aspx for more details.
    """
    import win32com.client  # pylint: disable=F0401
    wmi_service = win32com.client.Dispatch('WbemScripting.SWbemLocator')
    wbem = wmi_service.ConnectServer('.', 'root\\cimv2')
    return [proc for proc in wbem.ExecQuery('SELECT * FROM Win32_Process')]


  def _filter_processes_dir_win(processes, root_dir):
    """Returns all processes which has their main executable located inside
    root_dir.
    """
    def normalize_path(filename):
      try:
        return GetLongPathName(unicode(filename)).lower()
      except:  # pylint: disable=W0702
        return unicode(filename).lower()

    root_dir = normalize_path(root_dir)

    def process_name(proc):
      if proc.ExecutablePath:
        return normalize_path(proc.ExecutablePath)
      # proc.ExecutablePath may be empty if the process hasn't finished
      # initializing, but the command line may be valid.
      if proc.CommandLine is None:
        return None
      parsed_line = shlex.split(proc.CommandLine)
      if len(parsed_line) >= 1 and os.path.isabs(parsed_line[0]):
        return normalize_path(parsed_line[0])
      return None

    long_names = ((process_name(proc), proc) for proc in processes)

    return [
      proc for name, proc in long_names
      if name is not None and name.startswith(root_dir)
    ]


  def _filter_processes_tree_win(processes):
    """Returns all the processes under the current process."""
    # Convert to dict.
    processes = dict((p.ProcessId, p) for p in processes)
    root_pid = os.getpid()
    out = {root_pid: processes[root_pid]}
    while True:
      found = set()
      for pid in out:
        found.update(
            p.ProcessId for p in processes.itervalues()
            if p.ParentProcessId == pid)
      found -= set(out)
      if not found:
        break
      out.update((p, processes[p]) for p in found)
    return out.values()


  def _get_children_processes_win(root):
    """Returns a list of processes.

    Enumerates both:
    - all child processes from this process.
    - processes where the main executable in inside 'root'. The reason is that
      the ancestry may be broken so stray grand-children processes could be
      undetected by the first technique.

    This technique is not fool-proof but gets mostly there.
    """
    processes = _enum_processes_win()
    tree_processes = _filter_processes_tree_win(processes)
    dir_processes = _filter_processes_dir_win(processes, root)
    # Convert to dict to remove duplicates.
    processes = dict((p.ProcessId, p) for p in tree_processes)
    processes.update((p.ProcessId, p) for p in dir_processes)
    processes.pop(os.getpid())
    return processes


elif sys.platform == 'darwin':


  # On non-windows, keep the stdlib behavior.
  isabs = os.path.isabs


  def find_item_native_case(root_path, item):
    """Gets the native path case of a single item based at root_path.

    There is no API to get the native path case of symlinks on OSX. So it
    needs to be done the slow way.
    """
    if item == '..':
      return item

    item = item.lower()
    for element in fs.listdir(root_path):
      if element.lower() == item:
        return element


  @tools.profile
  @tools.cached
  def get_native_path_case(path):
    """Returns the native path case for an existing file.

    Technically, it's only HFS+ on OSX that is case preserving and
    insensitive. It's the default setting on HFS+ but can be changed.
    """
    assert isinstance(path, unicode), repr(path)
    if not isabs(path):
      raise ValueError(
          'get_native_path_case(%r): Require an absolute path' % path, path)
    if path.startswith('/dev'):
      # /dev is not visible from Carbon, causing an exception.
      return path

    # Starts assuming there is no symlink along the path.
    resolved = _native_case(path)
    if path.lower() in (resolved.lower(), resolved.lower() + './'):
      # This code path is incredibly faster.
      #logging.debug('get_native_path_case(%s) = %s' % (path, resolved))
      return resolved

    # There was a symlink, process it.
    base, symlink, rest = _split_at_symlink_native(None, path)
    if not symlink:
      # TODO(maruel): This can happen on OSX because we use stale APIs on OSX.
      # Fixing the APIs usage will likely fix this bug. The bug occurs due to
      # hardlinked files, where the API may return one file path or the other
      # depending on how it feels.
      return base
    prev = base
    base = safe_join(_native_case(base), symlink)
    assert len(base) > len(prev)
    while rest:
      prev = base
      relbase, symlink, rest = _split_at_symlink_native(base, rest)
      base = safe_join(base, relbase)
      assert len(base) > len(prev), (prev, base, symlink)
      if symlink:
        base = safe_join(base, symlink)
      assert len(base) > len(prev), (prev, base, symlink)
    # Make sure no symlink was resolved.
    assert base.lower() == path.lower(), (base, path)
    #logging.debug('get_native_path_case(%s) = %s' % (path, base))
    return base


  def enable_symlink():
    return True


  ## OSX private code.


  def _native_case(p):
    """Gets the native path case. Warning: this function resolves symlinks."""
    try:
      rel_ref, _ = Carbon.File.FSPathMakeRef(p.encode('utf-8'))
      # The OSX underlying code uses NFD but python strings are in NFC. This
      # will cause issues with os.listdir() for example. Since the dtrace log
      # *is* in NFC, normalize it here.
      out = unicodedata.normalize(
          'NFC', rel_ref.FSRefMakePath().decode('utf-8'))
      if p.endswith(os.path.sep) and not out.endswith(os.path.sep):
        return out + os.path.sep
      return out
    except MacOS.Error, e:
      if e.args[0] in (-43, -120):
        # The path does not exist. Try to recurse and reconstruct the path.
        # -43 means file not found.
        # -120 means directory not found.
        base = os.path.dirname(p)
        rest = os.path.basename(p)
        return os.path.join(_native_case(base), rest)
      raise OSError(
          e.args[0], 'Failed to get native path for %s' % p, p, e.args[1])


  def _split_at_symlink_native(base_path, rest):
    """Returns the native path for a symlink."""
    base, symlink, rest = split_at_symlink(base_path, rest)
    if symlink:
      if not base_path:
        base_path = base
      else:
        base_path = safe_join(base_path, base)
      symlink = find_item_native_case(base_path, symlink)
    return base, symlink, rest


else:  # OSes other than Windows and OSX.


  # On non-windows, keep the stdlib behavior.
  isabs = os.path.isabs


  def find_item_native_case(root, item):
    """Gets the native path case of a single item based at root_path."""
    if item == '..':
      return item

    root = get_native_path_case(root)
    return os.path.basename(get_native_path_case(os.path.join(root, item)))


  @tools.profile
  @tools.cached
  def get_native_path_case(path):
    """Returns the native path case for an existing file.

    On OSes other than OSX and Windows, assume the file system is
    case-sensitive.

    TODO(maruel): This is not strictly true. Implement if necessary.
    """
    assert isinstance(path, unicode), repr(path)
    if not isabs(path):
      raise ValueError(
          'get_native_path_case(%r): Require an absolute path' % path, path)
    # Give up on cygwin, as GetLongPathName() can't be called.
    # Linux traces tends to not be normalized so use this occasion to normalize
    # it. This function implementation already normalizes the path on the other
    # OS so this needs to be done here to be coherent between OSes.
    out = os.path.normpath(path)
    if path.endswith(os.path.sep) and not out.endswith(os.path.sep):
      out = out + os.path.sep
    # In 99.99% of cases on Linux out == path. Since a return value is cached
    # forever, reuse (also cached) |path| object. It safes approx 7MB of ram
    # when isolating Chromium tests. It's important on memory constrained
    # systems running ARM.
    return path if out == path else out


  def enable_symlink():
    return True


if sys.platform != 'win32':  # All non-Windows OSes.


  def safe_join(*args):
    """Joins path elements like os.path.join() but doesn't abort on absolute
    path.

    os.path.join('foo', '/bar') == '/bar'
    but safe_join('foo', '/bar') == 'foo/bar'.
    """
    out = ''
    for element in args:
      if element.startswith(os.path.sep):
        if out.endswith(os.path.sep):
          out += element[1:]
        else:
          out += element
      else:
        if out.endswith(os.path.sep):
          out += element
        else:
          out += os.path.sep + element
    return out


  @tools.profile
  def split_at_symlink(base_dir, relfile):
    """Scans each component of relfile and cut the string at the symlink if
    there is any.

    Returns a tuple (base_path, symlink, rest), with symlink == rest == None if
    not symlink was found.
    """
    if base_dir:
      assert relfile
      assert os.path.isabs(base_dir)
      index = 0
    else:
      assert os.path.isabs(relfile)
      index = 1

    def at_root(rest):
      if base_dir:
        return safe_join(base_dir, rest)
      return rest

    while True:
      try:
        index = relfile.index(os.path.sep, index)
      except ValueError:
        index = len(relfile)
      full = at_root(relfile[:index])
      if fs.islink(full):
        # A symlink!
        base = os.path.dirname(relfile[:index])
        symlink = os.path.basename(relfile[:index])
        rest = relfile[index:]
        logging.debug(
            'split_at_symlink(%s, %s) -> (%s, %s, %s)' %
            (base_dir, relfile, base, symlink, rest))
        return base, symlink, rest
      if index == len(relfile):
        break
      index += 1
    return relfile, None, None


  def kill_children_processes(root):
    """Not yet implemented on posix."""
    # pylint: disable=unused-argument
    return False


def relpath(path, root):
  """os.path.relpath() that keeps trailing os.path.sep."""
  out = os.path.relpath(path, root)
  if path.endswith(os.path.sep):
    out += os.path.sep
  return out


def safe_relpath(filepath, basepath):
  """Do not throw on Windows when filepath and basepath are on different drives.

  Different than relpath() above since this one doesn't keep the trailing
  os.path.sep and it swallows exceptions on Windows and return the original
  absolute path in the case of different drives.
  """
  try:
    return os.path.relpath(filepath, basepath)
  except ValueError:
    assert sys.platform == 'win32'
    return filepath


def normpath(path):
  """os.path.normpath() that keeps trailing os.path.sep."""
  out = os.path.normpath(path)
  if path.endswith(os.path.sep):
    out += os.path.sep
  return out


def posix_relpath(path, root):
  """posix.relpath() that keeps trailing slash.

  It is different from relpath() since it can be used on Windows.
  """
  out = posixpath.relpath(path, root)
  if path.endswith('/'):
    out += '/'
  return out


def is_url(path):
  """Returns True if it looks like an HTTP url instead of a file path."""
  return bool(re.match(r'^https?://.+$', path))


def path_starts_with(prefix, path):
  """Returns true if the components of the path |prefix| are the same as the
  initial components of |path| (or all of the components of |path|). The paths
  must be absolute.
  """
  #assert os.path.isabs(prefix) and os.path.isabs(path)
  prefix = os.path.normpath(prefix)
  path = os.path.normpath(path)
  #assert prefix == get_native_path_case(prefix), prefix
  #assert path == get_native_path_case(path), path
  prefix = prefix.rstrip(os.path.sep) + os.path.sep
  path = path.rstrip(os.path.sep) + os.path.sep
  return path.startswith(prefix)


@tools.profile
def fix_native_path_case(root, path):
  """Ensures that each component of |path| has the proper native case.

  It does so by iterating slowly over the directory elements of |path|. The file
  must exist.
  """
  native_case_path = root
  for raw_part in path.split(os.sep):
    if not raw_part or raw_part == '.':
      break

    part = find_item_native_case(native_case_path, raw_part)
    if not part:
      raise OSError(
          'File %s doesn\'t exist' %
          os.path.join(native_case_path, raw_part))
    native_case_path = os.path.join(native_case_path, part)

  return os.path.normpath(native_case_path)


def ensure_command_has_abs_path(command, cwd):
  """Ensures that an isolate command uses absolute path.

  This is needed since isolate can specify a command relative to 'cwd' and
  subprocess.call doesn't consider 'cwd' when searching for executable.
  """
  if not os.path.isabs(command[0]):
    command[0] = os.path.abspath(os.path.join(cwd, command[0]))


def is_same_filesystem(path1, path2):
  """Returns True if both paths are on the same filesystem.

  This is required to enable the use of hardlinks.
  """
  assert os.path.isabs(path1), path1
  assert os.path.isabs(path2), path2
  if sys.platform == 'win32':
    # If the drive letter mismatches, assume it's a separate partition.
    # TODO(maruel): It should look at the underlying drive, a drive letter could
    # be a mount point to a directory on another drive.
    assert re.match(ur'^[a-zA-Z]\:\\.*', path1), path1
    assert re.match(ur'^[a-zA-Z]\:\\.*', path2), path2
    if path1[0].lower() != path2[0].lower():
      return False
  return fs.stat(path1).st_dev == fs.stat(path2).st_dev


def get_free_space(path):
  """Returns the number of free bytes.

  On POSIX platforms, this returns the free space as visible by the current
  user. On some systems, there's a percentage of the free space on the partition
  that is only accessible as the root user.
  """
  if sys.platform == 'win32':
    free_bytes = ctypes.c_ulonglong(0)
    windll.kernel32.GetDiskFreeSpaceExW(
        ctypes.c_wchar_p(path), None, None, ctypes.pointer(free_bytes))
    return free_bytes.value
  # For OSes other than Windows.
  f = fs.statvfs(path)  # pylint: disable=E1101
  return f.f_bfree * f.f_frsize


### Write file functions.


def hardlink(source, link_name):
  """Hardlinks a file.

  Add support for os.link() on Windows.
  """
  assert isinstance(source, unicode), source
  assert isinstance(link_name, unicode), link_name
  if sys.platform == 'win32':
    if not windll.kernel32.CreateHardLinkW(
        fs.extend(link_name), fs.extend(source), 0):
      raise OSError()
  else:
    fs.link(source, link_name)


def readable_copy(outfile, infile):
  """Makes a copy of the file that is readable by everyone."""
  fs.copy2(infile, outfile)
  fs.chmod(
      outfile,
      fs.stat(outfile).st_mode | stat.S_IRUSR | stat.S_IRGRP | stat.S_IROTH)


def set_read_only(path, read_only):
  """Sets or resets the write bit on a file or directory.

  Zaps out access to 'group' and 'others'.
  """
  orig_mode = fs.lstat(path).st_mode
  mode = orig_mode
  # TODO(maruel): Stop removing GO bits.
  if read_only:
    mode &= stat.S_IRUSR|stat.S_IXUSR # 0500
  else:
    mode |= stat.S_IRUSR|stat.S_IWUSR # 0600
    if sys.platform != 'win32' and stat.S_ISDIR(mode):
      mode |= stat.S_IXUSR # 0100
  if hasattr(os, 'lchmod'):
    fs.lchmod(path, mode)  # pylint: disable=E1101
  else:
    if stat.S_ISLNK(orig_mode):
      # Skip symlink without lchmod() support.
      return

    # TODO(maruel): Implement proper DACL modification on Windows.
    fs.chmod(path, mode)


def set_read_only_swallow(path, read_only):
  """Returns if an OSError exception occured."""
  try:
    set_read_only(path, read_only)
  except OSError as e:
    return e


def remove(filepath):
  """Removes a file, even if it is read-only."""
  # TODO(maruel): Not do it unless necessary since it slows this function
  # down.
  if sys.platform == 'win32':
    # Deleting a read-only file will fail if it is read-only.
    set_read_only(filepath, False)
  else:
    # Deleting a read-only file will fail if the directory is read-only.
    set_read_only(os.path.dirname(filepath), False)
  fs.remove(filepath)


def try_remove(filepath):
  """Removes a file without crashing even if it doesn't exist.

  Returns:
    True if the removal successed.
  """
  try:
    remove(filepath)
    return True
  except OSError:
    return False


def link_file(outfile, infile, action):
  """Links a file. The type of link depends on |action|.

  Returns:
    True if the action was carried on, False if fallback was used.
  """
  if action < 1 or action > COPY:
    raise ValueError('Unknown mapping action %s' % action)
  # TODO(maruel): Skip these checks.
  if not fs.isfile(infile):
    raise OSError('%s is missing' % infile)
  if fs.isfile(outfile):
    raise OSError(
        '%s already exist; insize:%d; outsize:%d' %
        (outfile, fs.stat(infile).st_size, fs.stat(outfile).st_size))

  if action == COPY:
    readable_copy(outfile, infile)
    return True

  if action in (SYMLINK, SYMLINK_WITH_FALLBACK):
    try:
      fs.symlink(infile, outfile)  # pylint: disable=E1101
      return True
    except OSError:
      if action == SYMLINK:
        raise
      logging.warning(
          'Failed to symlink, falling back to copy %s to %s' % (
            infile, outfile))
      # Signal caller that fallback copy was used.
      readable_copy(outfile, infile)
      return False

  # HARDLINK or HARDLINK_WITH_FALLBACK.
  try:
    hardlink(infile, outfile)
    return True
  except OSError as e:
    if action == HARDLINK:
      raise OSError('Failed to hardlink %s to %s: %s' % (infile, outfile, e))

  # Probably a different file system.
  logging.warning(
      'Failed to hardlink, falling back to copy %s to %s' % (
        infile, outfile))
  readable_copy(outfile, infile)
  # Signal caller that fallback copy was used.
  return False


def atomic_replace(path, body):
  """Atomically replaces content of the file at given path.

  'body' will be written to the file as is (as in open(..., 'wb') mode).

  Does not preserve file attributes.

  Raises OSError or IOError on errors (e.g. in case the file is locked on
  Windows). The caller may retry a bunch of times in such cases before giving
  up.
  """
  assert path and path[-1] != os.sep, path
  path = os.path.abspath(path)
  dir_name, base_name = os.path.split(path)

  fd, tmp_name = tempfile.mkstemp(dir=dir_name, prefix=base_name+'_')
  try:
    with os.fdopen(fd, 'wb') as f:
      f.write(body)
      f.flush()
      os.fsync(fd)
    if sys.platform != 'win32':
      os.rename(tmp_name, path)
    else:
      # Flags are MOVEFILE_REPLACE_EXISTING|MOVEFILE_WRITE_THROUGH.
      MoveFileEx(unicode(tmp_name), unicode(path), 0x1|0x8)
    tmp_name =  None # no need to remove it in 'finally' block anymore
  finally:
    if tmp_name:
      try:
        os.remove(tmp_name)
      except OSError as e:
        logging.warning(
            'Failed to delete temp file %s in replace_file_content: %s',
            tmp_name, e)


### Write directory functions.


def ensure_tree(path, perm=0777):
  """Ensures a directory exists."""
  if not fs.isdir(path):
    fs.makedirs(path, perm)


def make_tree_read_only(root):
  """Makes all the files in the directories read only.

  Also makes the directories read only, only if it makes sense on the platform.

  This means no file can be created or deleted.
  """
  err = None
  logging.debug('make_tree_read_only(%s)', root)
  for dirpath, dirnames, filenames in fs.walk(root, topdown=True):
    for filename in filenames:
      e = set_read_only_swallow(os.path.join(dirpath, filename), True)
      if not err:
        err = e
    if sys.platform != 'win32':
      # It must not be done on Windows.
      for dirname in dirnames:
        e = set_read_only_swallow(os.path.join(dirpath, dirname), True)
        if not err:
          err = e
  if sys.platform != 'win32':
    e = set_read_only_swallow(root, True)
    if not err:
      err = e
  if err:
    # pylint: disable=raising-bad-type
    raise err


def make_tree_files_read_only(root):
  """Makes all the files in the directories read only but not the directories
  themselves.

  This means files can be created or deleted.
  """
  logging.debug('make_tree_files_read_only(%s)', root)
  if sys.platform != 'win32':
    set_read_only(root, False)
  for dirpath, dirnames, filenames in fs.walk(root, topdown=True):
    for filename in filenames:
      set_read_only(os.path.join(dirpath, filename), True)
    if sys.platform != 'win32':
      # It must not be done on Windows.
      for dirname in dirnames:
        set_read_only(os.path.join(dirpath, dirname), False)


def make_tree_writeable(root):
  """Makes all the files in the directories writeable.

  Also makes the directories writeable, only if it makes sense on the platform.

  It is different from make_tree_deleteable() because it unconditionally affects
  the files.
  """
  logging.debug('make_tree_writeable(%s)', root)
  if sys.platform != 'win32':
    set_read_only(root, False)
  for dirpath, dirnames, filenames in fs.walk(root, topdown=True):
    for filename in filenames:
      set_read_only(os.path.join(dirpath, filename), False)
    if sys.platform != 'win32':
      # It must not be done on Windows.
      for dirname in dirnames:
        set_read_only(os.path.join(dirpath, dirname), False)


def make_tree_deleteable(root):
  """Changes the appropriate permissions so the files in the directories can be
  deleted.

  On Windows, the files are modified. On other platforms, modify the directory.
  It only does the minimum so the files can be deleted safely.

  Warning on Windows: since file permission is modified, the file node is
  modified. This means that for hard-linked files, every directory entry for the
  file node has its file permission modified.
  """
  logging.debug('make_tree_deleteable(%s)', root)
  err = None
  sudo_failed = False

  def try_sudo(p):
    if sys.platform == 'linux2' and not sudo_failed:
      # Try passwordless sudo, just in case. In practice, it is preferable
      # to use linux capabilities.
      with open(os.devnull, 'rb') as f:
        if not subprocess42.call(
            ['sudo', '-n', 'chmod', 'a+rwX,-t', p], stdin=f):
          return False
      logging.debug('sudo chmod %s failed', p)
    return True

  if sys.platform != 'win32':
    e = set_read_only_swallow(root, False)
    if e:
      sudo_failed = try_sudo(root)
    if not err:
      err = e
  for dirpath, dirnames, filenames in fs.walk(root, topdown=True):
    if sys.platform == 'win32':
      for filename in filenames:
        e = set_read_only_swallow(os.path.join(dirpath, filename), False)
        if not err:
          err = e
    else:
      for dirname in dirnames:
        p = os.path.join(dirpath, dirname)
        e = set_read_only_swallow(p, False)
        if e:
          sudo_failed = try_sudo(p)
        if not err:
          err = e
  if err:
    # pylint: disable=raising-bad-type
    raise err


def rmtree(root):
  """Wrapper around shutil.rmtree() to retry automatically on Windows.

  On Windows, forcibly kills processes that are found to interfere with the
  deletion.

  Returns:
    True on normal execution, False if berserk techniques (like killing
    processes) had to be used.
  """
  logging.info('rmtree(%s)', root)
  assert isinstance(root, unicode) or sys.getdefaultencoding() == 'utf-8', (
      repr(root), sys.getdefaultencoding())
  root = unicode(root)
  try:
    make_tree_deleteable(root)
  except OSError as e:
    logging.warning('Swallowing make_tree_deleteable() error: %s', e)

  # First try the soft way: tries 3 times to delete and sleep a bit in between.
  # Retries help if test subprocesses outlive main process and try to actively
  # use or write to the directory while it is being deleted.
  max_tries = 3
  for i in xrange(max_tries):
    # errors is a list of tuple(function, path, excinfo).
    errors = []
    fs.rmtree(root, onerror=lambda *args: errors.append(args))
    if not errors or not fs.exists(root):
      if i:
        sys.stderr.write('Succeeded.\n')
      return True
    if not i and sys.platform == 'win32':
      for path in sorted(set(path for _, path, _ in errors)):
        try:
          change_acl_for_delete(path)
        except Exception as e:
          sys.stderr.write('- %s (failed to update ACL: %s)\n' % (path, e))

    if i != max_tries - 1:
      delay = (i+1)*2
      sys.stderr.write(
          'Failed to delete %s (%d files remaining).\n'
          '  Maybe the test has a subprocess outliving it.\n'
          '  Sleeping %d seconds.\n' %
          (root, len(errors), delay))
      time.sleep(delay)

  sys.stderr.write(
      'Failed to delete %s. The following files remain:\n' % root)
  # The same path may be listed multiple times.
  for path in sorted(set(path for _, path, _ in errors)):
    sys.stderr.write('- %s\n' % path)

  # If soft retries fail on Linux, there's nothing better we can do.
  if sys.platform != 'win32':
    raise errors[0][2][0], errors[0][2][1], errors[0][2][2]

  # The soft way was not good enough. Try the hard way.
  for i in xrange(max_tries):
    if not kill_children_processes(root):
      break
    if i != max_tries - 1:
      time.sleep((i+1)*2)
  else:
    processes = _get_children_processes_win(root)
    if processes:
      sys.stderr.write('Failed to terminate processes.\n')
      raise errors[0][2][0], errors[0][2][1], errors[0][2][2]

  # Now that annoying processes in root are evicted, try again.
  errors = []
  fs.rmtree(root, onerror=lambda *args: errors.append(args))
  if errors and fs.exists(root):
    # There's no hope: the directory was tried to be removed 4 times. Give up
    # and raise an exception.
    sys.stderr.write(
        'Failed to delete %s. The following files remain:\n' % root)
    # The same path may be listed multiple times.
    for path in sorted(set(path for _, path, _ in errors)):
      sys.stderr.write('- %s\n' % path)
    raise errors[0][2][0], errors[0][2][1], errors[0][2][2]
  return False


## Private code.
