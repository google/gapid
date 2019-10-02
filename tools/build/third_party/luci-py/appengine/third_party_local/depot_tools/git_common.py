# Copyright 2014 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Monkeypatch IMapIterator so that Ctrl-C can kill everything properly.
# Derived from https://gist.github.com/aljungberg/626518
import multiprocessing.pool
from multiprocessing.pool import IMapIterator
def wrapper(func):
  def wrap(self, timeout=None):
    return func(self, timeout=timeout or 1e100)
  return wrap
IMapIterator.next = wrapper(IMapIterator.next)
IMapIterator.__next__ = IMapIterator.next
# TODO(iannucci): Monkeypatch all other 'wait' methods too.


import binascii
import collections
import contextlib
import functools
import logging
import os
import re
import signal
import sys
import tempfile
import textwrap
import threading

import subprocess2

ROOT = os.path.abspath(os.path.dirname(__file__))

GIT_EXE = ROOT+'\\git.bat' if sys.platform.startswith('win') else 'git'
TEST_MODE = False

FREEZE = 'FREEZE'
FREEZE_SECTIONS = {
  'indexed': 'soft',
  'unindexed': 'mixed'
}
FREEZE_MATCHER = re.compile(r'%s.(%s)' % (FREEZE, '|'.join(FREEZE_SECTIONS)))


# Retry a git operation if git returns a error response with any of these
# messages. It's all observed 'bad' GoB responses so far.
#
# This list is inspired/derived from the one in ChromiumOS's Chromite:
# <CHROMITE>/lib/git.py::GIT_TRANSIENT_ERRORS
#
# It was last imported from '7add3ac29564d98ac35ce426bc295e743e7c0c02'.
GIT_TRANSIENT_ERRORS = (
    # crbug.com/285832
    r'!.*\[remote rejected\].*\(error in hook\)',

    # crbug.com/289932
    r'!.*\[remote rejected\].*\(failed to lock\)',

    # crbug.com/307156
    r'!.*\[remote rejected\].*\(error in Gerrit backend\)',

    # crbug.com/285832
    r'remote error: Internal Server Error',

    # crbug.com/294449
    r'fatal: Couldn\'t find remote ref ',

    # crbug.com/220543
    r'git fetch_pack: expected ACK/NAK, got',

    # crbug.com/189455
    r'protocol error: bad pack header',

    # crbug.com/202807
    r'The remote end hung up unexpectedly',

    # crbug.com/298189
    r'TLS packet with unexpected length was received',

    # crbug.com/187444
    r'RPC failed; result=\d+, HTTP code = \d+',

    # crbug.com/388876
    r'Connection timed out',

    # crbug.com/430343
    # TODO(dnj): Resync with Chromite.
    r'The requested URL returned error: 5\d+',
)

GIT_TRANSIENT_ERRORS_RE = re.compile('|'.join(GIT_TRANSIENT_ERRORS),
                                     re.IGNORECASE)

# git's for-each-ref command first supported the upstream:track token in its
# format string in version 1.9.0, but some usages were broken until 2.3.0.
# See git commit b6160d95 for more information.
MIN_UPSTREAM_TRACK_GIT_VERSION = (2, 3)

class BadCommitRefException(Exception):
  def __init__(self, refs):
    msg = ('one of %s does not seem to be a valid commitref.' %
           str(refs))
    super(BadCommitRefException, self).__init__(msg)


def memoize_one(**kwargs):
  """Memoizes a single-argument pure function.

  Values of None are not cached.

  Kwargs:
    threadsafe (bool) - REQUIRED. Specifies whether to use locking around
      cache manipulation functions. This is a kwarg so that users of memoize_one
      are forced to explicitly and verbosely pick True or False.

  Adds three methods to the decorated function:
    * get(key, default=None) - Gets the value for this key from the cache.
    * set(key, value) - Sets the value for this key from the cache.
    * clear() - Drops the entire contents of the cache.  Useful for unittests.
    * update(other) - Updates the contents of the cache from another dict.
  """
  assert 'threadsafe' in kwargs, 'Must specify threadsafe={True,False}'
  threadsafe = kwargs['threadsafe']

  if threadsafe:
    def withlock(lock, f):
      def inner(*args, **kwargs):
        with lock:
          return f(*args, **kwargs)
      return inner
  else:
    def withlock(_lock, f):
      return f

  def decorator(f):
    # Instantiate the lock in decorator, in case users of memoize_one do:
    #
    # memoizer = memoize_one(threadsafe=True)
    #
    # @memoizer
    # def fn1(val): ...
    #
    # @memoizer
    # def fn2(val): ...

    lock = threading.Lock() if threadsafe else None
    cache = {}
    _get = withlock(lock, cache.get)
    _set = withlock(lock, cache.__setitem__)

    @functools.wraps(f)
    def inner(arg):
      ret = _get(arg)
      if ret is None:
        ret = f(arg)
        if ret is not None:
          _set(arg, ret)
      return ret
    inner.get = _get
    inner.set = _set
    inner.clear = withlock(lock, cache.clear)
    inner.update = withlock(lock, cache.update)
    return inner
  return decorator


def _ScopedPool_initer(orig, orig_args):  # pragma: no cover
  """Initializer method for ScopedPool's subprocesses.

  This helps ScopedPool handle Ctrl-C's correctly.
  """
  signal.signal(signal.SIGINT, signal.SIG_IGN)
  if orig:
    orig(*orig_args)


@contextlib.contextmanager
def ScopedPool(*args, **kwargs):
  """Context Manager which returns a multiprocessing.pool instance which
  correctly deals with thrown exceptions.

  *args - Arguments to multiprocessing.pool

  Kwargs:
    kind ('threads', 'procs') - The type of underlying coprocess to use.
    **etc - Arguments to multiprocessing.pool
  """
  if kwargs.pop('kind', None) == 'threads':
    pool = multiprocessing.pool.ThreadPool(*args, **kwargs)
  else:
    orig, orig_args = kwargs.get('initializer'), kwargs.get('initargs', ())
    kwargs['initializer'] = _ScopedPool_initer
    kwargs['initargs'] = orig, orig_args
    pool = multiprocessing.pool.Pool(*args, **kwargs)

  try:
    yield pool
    pool.close()
  except:
    pool.terminate()
    raise
  finally:
    pool.join()


class ProgressPrinter(object):
  """Threaded single-stat status message printer."""
  def __init__(self, fmt, enabled=None, fout=sys.stderr, period=0.5):
    """Create a ProgressPrinter.

    Use it as a context manager which produces a simple 'increment' method:

      with ProgressPrinter('(%%(count)d/%d)' % 1000) as inc:
        for i in xrange(1000):
          # do stuff
          if i % 10 == 0:
            inc(10)

    Args:
      fmt - String format with a single '%(count)d' where the counter value
        should go.
      enabled (bool) - If this is None, will default to True if
        logging.getLogger() is set to INFO or more verbose.
      fout (file-like) - The stream to print status messages to.
      period (float) - The time in seconds for the printer thread to wait
        between printing.
    """
    self.fmt = fmt
    if enabled is None:  # pragma: no cover
      self.enabled = logging.getLogger().isEnabledFor(logging.INFO)
    else:
      self.enabled = enabled

    self._count = 0
    self._dead = False
    self._dead_cond = threading.Condition()
    self._stream = fout
    self._thread = threading.Thread(target=self._run)
    self._period = period

  def _emit(self, s):
    if self.enabled:
      self._stream.write('\r' + s)
      self._stream.flush()

  def _run(self):
    with self._dead_cond:
      while not self._dead:
        self._emit(self.fmt % {'count': self._count})
        self._dead_cond.wait(self._period)
        self._emit((self.fmt + '\n') % {'count': self._count})

  def inc(self, amount=1):
    self._count += amount

  def __enter__(self):
    self._thread.start()
    return self.inc

  def __exit__(self, _exc_type, _exc_value, _traceback):
    self._dead = True
    with self._dead_cond:
      self._dead_cond.notifyAll()
    self._thread.join()
    del self._thread


def once(function):
  """@Decorates |function| so that it only performs its action once, no matter
  how many times the decorated |function| is called."""
  def _inner_gen():
    yield function()
    while True:
      yield
  return _inner_gen().next


## Git functions


def branch_config(branch, option, default=None):
  return config('branch.%s.%s' % (branch, option), default=default)


def branch_config_map(option):
  """Return {branch: <|option| value>} for all branches."""
  try:
    reg = re.compile(r'^branch\.(.*)\.%s$' % option)
    lines = run('config', '--get-regexp', reg.pattern).splitlines()
    return {reg.match(k).group(1): v for k, v in (l.split() for l in lines)}
  except subprocess2.CalledProcessError:
    return {}


def branches(*args):
  NO_BRANCH = ('* (no branch', '* (detached', '* (HEAD detached')

  key = 'depot-tools.branch-limit'
  limit = 20
  try:
    limit = int(config(key, limit))
  except ValueError:
    pass

  raw_branches = run('branch', *args).splitlines()

  num = len(raw_branches)
  if num > limit:
    print >> sys.stderr, textwrap.dedent("""\
    Your git repo has too many branches (%d/%d) for this tool to work well.

    You may adjust this limit by running:
      git config %s <new_limit>
    """ % (num, limit, key))
    sys.exit(1)

  for line in raw_branches:
    if line.startswith(NO_BRANCH):
      continue
    yield line.split()[-1]


def config(option, default=None):
  try:
    return run('config', '--get', option) or default
  except subprocess2.CalledProcessError:
    return default


def config_list(option):
  try:
    return run('config', '--get-all', option).split()
  except subprocess2.CalledProcessError:
    return []


def current_branch():
  try:
    return run('rev-parse', '--abbrev-ref', 'HEAD')
  except subprocess2.CalledProcessError:
    return None


def del_branch_config(branch, option, scope='local'):
  del_config('branch.%s.%s' % (branch, option), scope=scope)


def del_config(option, scope='local'):
  try:
    run('config', '--' + scope, '--unset', option)
  except subprocess2.CalledProcessError:
    pass


def freeze():
  took_action = False

  try:
    run('commit', '--no-verify', '-m', FREEZE + '.indexed')
    took_action = True
  except subprocess2.CalledProcessError:
    pass

  try:
    run('add', '-A')
    run('commit', '--no-verify', '-m', FREEZE + '.unindexed')
    took_action = True
  except subprocess2.CalledProcessError:
    pass

  if not took_action:
    return 'Nothing to freeze.'


def get_branch_tree():
  """Get the dictionary of {branch: parent}, compatible with topo_iter.

  Returns a tuple of (skipped, <branch_tree dict>) where skipped is a set of
  branches without upstream branches defined.
  """
  skipped = set()
  branch_tree = {}

  for branch in branches():
    parent = upstream(branch)
    if not parent:
      skipped.add(branch)
      continue
    branch_tree[branch] = parent

  return skipped, branch_tree


def get_or_create_merge_base(branch, parent=None):
  """Finds the configured merge base for branch.

  If parent is supplied, it's used instead of calling upstream(branch).
  """
  base = branch_config(branch, 'base')
  base_upstream = branch_config(branch, 'base-upstream')
  parent = parent or upstream(branch)
  if parent is None or branch is None:
    return None
  actual_merge_base = run('merge-base', parent, branch)

  if base_upstream != parent:
    base = None
    base_upstream = None

  def is_ancestor(a, b):
    return run_with_retcode('merge-base', '--is-ancestor', a, b) == 0

  if base:
    if not is_ancestor(base, branch):
      logging.debug('Found WRONG pre-set merge-base for %s: %s', branch, base)
      base = None
    elif is_ancestor(base, actual_merge_base):
      logging.debug('Found OLD pre-set merge-base for %s: %s', branch, base)
      base = None
    else:
      logging.debug('Found pre-set merge-base for %s: %s', branch, base)

  if not base:
    base = actual_merge_base
    manual_merge_base(branch, base, parent)

  return base


def hash_multi(*reflike):
  return run('rev-parse', *reflike).splitlines()


def hash_one(reflike, short=False):
  args = ['rev-parse', reflike]
  if short:
    args.insert(1, '--short')
  return run(*args)


def in_rebase():
  git_dir = run('rev-parse', '--git-dir')
  return (
    os.path.exists(os.path.join(git_dir, 'rebase-merge')) or
    os.path.exists(os.path.join(git_dir, 'rebase-apply')))


def intern_f(f, kind='blob'):
  """Interns a file object into the git object store.

  Args:
    f (file-like object) - The file-like object to intern
    kind (git object type) - One of 'blob', 'commit', 'tree', 'tag'.

  Returns the git hash of the interned object (hex encoded).
  """
  ret = run('hash-object', '-t', kind, '-w', '--stdin', stdin=f)
  f.close()
  return ret


def is_dormant(branch):
  # TODO(iannucci): Do an oldness check?
  return branch_config(branch, 'dormant', 'false') != 'false'


def manual_merge_base(branch, base, parent):
  set_branch_config(branch, 'base', base)
  set_branch_config(branch, 'base-upstream', parent)


def mktree(treedict):
  """Makes a git tree object and returns its hash.

  See |tree()| for the values of mode, type, and ref.

  Args:
    treedict - { name: (mode, type, ref) }
  """
  with tempfile.TemporaryFile() as f:
    for name, (mode, typ, ref) in treedict.iteritems():
      f.write('%s %s %s\t%s\0' % (mode, typ, ref, name))
    f.seek(0)
    return run('mktree', '-z', stdin=f)


def parse_commitrefs(*commitrefs):
  """Returns binary encoded commit hashes for one or more commitrefs.

  A commitref is anything which can resolve to a commit. Popular examples:
    * 'HEAD'
    * 'origin/master'
    * 'cool_branch~2'
  """
  try:
    return map(binascii.unhexlify, hash_multi(*commitrefs))
  except subprocess2.CalledProcessError:
    raise BadCommitRefException(commitrefs)


RebaseRet = collections.namedtuple('RebaseRet', 'success stdout stderr')


def rebase(parent, start, branch, abort=False):
  """Rebases |start|..|branch| onto the branch |parent|.

  Args:
    parent - The new parent ref for the rebased commits.
    start  - The commit to start from
    branch - The branch to rebase
    abort  - If True, will call git-rebase --abort in the event that the rebase
             doesn't complete successfully.

  Returns a namedtuple with fields:
    success - a boolean indicating that the rebase command completed
              successfully.
    message - if the rebase failed, this contains the stdout of the failed
              rebase.
  """
  try:
    args = ['--onto', parent, start, branch]
    if TEST_MODE:
      args.insert(0, '--committer-date-is-author-date')
    run('rebase', *args)
    return RebaseRet(True, '', '')
  except subprocess2.CalledProcessError as cpe:
    if abort:
      run_with_retcode('rebase', '--abort')  # ignore failure
    return RebaseRet(False, cpe.stdout, cpe.stderr)


def remove_merge_base(branch):
  del_branch_config(branch, 'base')
  del_branch_config(branch, 'base-upstream')


def root():
  return config('depot-tools.upstream', 'origin/master')


def run(*cmd, **kwargs):
  """The same as run_with_stderr, except it only returns stdout."""
  return run_with_stderr(*cmd, **kwargs)[0]


def run_with_retcode(*cmd, **kwargs):
  """Run a command but only return the status code."""
  try:
    run(*cmd, **kwargs)
    return 0
  except subprocess2.CalledProcessError as cpe:
    return cpe.returncode


def run_stream(*cmd, **kwargs):
  """Runs a git command. Returns stdout as a PIPE (file-like object).

  stderr is dropped to avoid races if the process outputs to both stdout and
  stderr.
  """
  kwargs.setdefault('stderr', subprocess2.VOID)
  kwargs.setdefault('stdout', subprocess2.PIPE)
  cmd = (GIT_EXE, '-c', 'color.ui=never') + cmd
  proc = subprocess2.Popen(cmd, **kwargs)
  return proc.stdout


@contextlib.contextmanager
def run_stream_with_retcode(*cmd, **kwargs):
  """Runs a git command as context manager yielding stdout as a PIPE.

  stderr is dropped to avoid races if the process outputs to both stdout and
  stderr.

  Raises subprocess2.CalledProcessError on nonzero return code.
  """
  kwargs.setdefault('stderr', subprocess2.VOID)
  kwargs.setdefault('stdout', subprocess2.PIPE)
  cmd = (GIT_EXE, '-c', 'color.ui=never') + cmd
  try:
    proc = subprocess2.Popen(cmd, **kwargs)
    yield proc.stdout
  finally:
    retcode = proc.wait()
    if retcode != 0:
      raise subprocess2.CalledProcessError(retcode, cmd, os.getcwd(),
                                           None, None)


def run_with_stderr(*cmd, **kwargs):
  """Runs a git command.

  Returns (stdout, stderr) as a pair of strings.

  kwargs
    autostrip (bool) - Strip the output. Defaults to True.
    indata (str) - Specifies stdin data for the process.
  """
  kwargs.setdefault('stdin', subprocess2.PIPE)
  kwargs.setdefault('stdout', subprocess2.PIPE)
  kwargs.setdefault('stderr', subprocess2.PIPE)
  autostrip = kwargs.pop('autostrip', True)
  indata = kwargs.pop('indata', None)

  cmd = (GIT_EXE, '-c', 'color.ui=never') + cmd
  proc = subprocess2.Popen(cmd, **kwargs)
  ret, err = proc.communicate(indata)
  retcode = proc.wait()
  if retcode != 0:
    raise subprocess2.CalledProcessError(retcode, cmd, os.getcwd(), ret, err)

  if autostrip:
    ret = (ret or '').strip()
    err = (err or '').strip()

  return ret, err


def set_branch_config(branch, option, value, scope='local'):
  set_config('branch.%s.%s' % (branch, option), value, scope=scope)


def set_config(option, value, scope='local'):
  run('config', '--' + scope, option, value)


def get_dirty_files():
  # Make sure index is up-to-date before running diff-index.
  run_with_retcode('update-index', '--refresh', '-q')
  return run('diff-index', '--name-status', 'HEAD')


def is_dirty_git_tree(cmd):
  dirty = get_dirty_files()
  if dirty:
    print 'Cannot %s with a dirty tree. You must commit locally first.' % cmd
    print 'Uncommitted files: (git diff-index --name-status HEAD)'
    print dirty[:4096]
    if len(dirty) > 4096: # pragma: no cover
      print '... (run "git diff-index --name-status HEAD" to see full output).'
    return True
  return False


def squash_current_branch(header=None, merge_base=None):
  header = header or 'git squash commit.'
  merge_base = merge_base or get_or_create_merge_base(current_branch())
  log_msg = header + '\n'
  if log_msg:
    log_msg += '\n'
  log_msg += run('log', '--reverse', '--format=%H%n%B', '%s..HEAD' % merge_base)
  run('reset', '--soft', merge_base)

  if not get_dirty_files():
    # Sometimes the squash can result in the same tree, meaning that there is
    # nothing to commit at this point.
    print 'Nothing to commit; squashed branch is empty'
    return False
  run('commit', '--no-verify', '-a', '-F', '-', indata=log_msg)
  return True


def tags(*args):
  return run('tag', *args).splitlines()


def thaw():
  took_action = False
  for sha in (s.strip() for s in run_stream('rev-list', 'HEAD').xreadlines()):
    msg = run('show', '--format=%f%b', '-s', 'HEAD')
    match = FREEZE_MATCHER.match(msg)
    if not match:
      if not took_action:
        return 'Nothing to thaw.'
      break

    run('reset', '--' + FREEZE_SECTIONS[match.group(1)], sha)
    took_action = True


def topo_iter(branch_tree, top_down=True):
  """Generates (branch, parent) in topographical order for a branch tree.

  Given a tree:

            A1
        B1      B2
      C1  C2    C3
                D1

  branch_tree would look like: {
    'D1': 'C3',
    'C3': 'B2',
    'B2': 'A1',
    'C1': 'B1',
    'C2': 'B1',
    'B1': 'A1',
  }

  It is OK to have multiple 'root' nodes in your graph.

  if top_down is True, items are yielded from A->D. Otherwise they're yielded
  from D->A. Within a layer the branches will be yielded in sorted order.
  """
  branch_tree = branch_tree.copy()

  # TODO(iannucci): There is probably a more efficient way to do these.
  if top_down:
    while branch_tree:
      this_pass = [(b, p) for b, p in branch_tree.iteritems()
                   if p not in branch_tree]
      assert this_pass, "Branch tree has cycles: %r" % branch_tree
      for branch, parent in sorted(this_pass):
        yield branch, parent
        del branch_tree[branch]
  else:
    parent_to_branches = collections.defaultdict(set)
    for branch, parent in branch_tree.iteritems():
      parent_to_branches[parent].add(branch)

    while branch_tree:
      this_pass = [(b, p) for b, p in branch_tree.iteritems()
                   if not parent_to_branches[b]]
      assert this_pass, "Branch tree has cycles: %r" % branch_tree
      for branch, parent in sorted(this_pass):
        yield branch, parent
        parent_to_branches[parent].discard(branch)
        del branch_tree[branch]


def tree(treeref, recurse=False):
  """Returns a dict representation of a git tree object.

  Args:
    treeref (str) - a git ref which resolves to a tree (commits count as trees).
    recurse (bool) - include all of the tree's decendants too. File names will
      take the form of 'some/path/to/file'.

  Return format:
    { 'file_name': (mode, type, ref) }

    mode is an integer where:
      * 0040000 - Directory
      * 0100644 - Regular non-executable file
      * 0100664 - Regular non-executable group-writeable file
      * 0100755 - Regular executable file
      * 0120000 - Symbolic link
      * 0160000 - Gitlink

    type is a string where it's one of 'blob', 'commit', 'tree', 'tag'.

    ref is the hex encoded hash of the entry.
  """
  ret = {}
  opts = ['ls-tree', '--full-tree']
  if recurse:
    opts.append('-r')
  opts.append(treeref)
  try:
    for line in run(*opts).splitlines():
      mode, typ, ref, name = line.split(None, 3)
      ret[name] = (mode, typ, ref)
  except subprocess2.CalledProcessError:
    return None
  return ret


def upstream(branch):
  try:
    return run('rev-parse', '--abbrev-ref', '--symbolic-full-name',
               branch+'@{upstream}')
  except subprocess2.CalledProcessError:
    return None


def get_git_version():
  """Returns a tuple that contains the numeric components of the current git
  version."""
  version_string = run('--version')
  version_match = re.search(r'(\d+.)+(\d+)', version_string)
  version = version_match.group() if version_match else ''

  return tuple(int(x) for x in version.split('.'))


def get_branches_info(include_tracking_status):
  format_string = (
      '--format=%(refname:short):%(objectname:short):%(upstream:short):')

  # This is not covered by the depot_tools CQ which only has git version 1.8.
  if (include_tracking_status and
      get_git_version() >= MIN_UPSTREAM_TRACK_GIT_VERSION):  # pragma: no cover
    format_string += '%(upstream:track)'

  info_map = {}
  data = run('for-each-ref', format_string, 'refs/heads')
  BranchesInfo = collections.namedtuple(
      'BranchesInfo', 'hash upstream ahead behind')
  for line in data.splitlines():
    (branch, branch_hash, upstream_branch, tracking_status) = line.split(':')

    ahead_match = re.search(r'ahead (\d+)', tracking_status)
    ahead = int(ahead_match.group(1)) if ahead_match else None

    behind_match = re.search(r'behind (\d+)', tracking_status)
    behind = int(behind_match.group(1)) if behind_match else None

    info_map[branch] = BranchesInfo(
        hash=branch_hash, upstream=upstream_branch, ahead=ahead, behind=behind)

  # Set None for upstreams which are not branches (e.g empty upstream, remotes
  # and deleted upstream branches).
  missing_upstreams = {}
  for info in info_map.values():
    if info.upstream not in info_map and info.upstream not in missing_upstreams:
      missing_upstreams[info.upstream] = None

  return dict(info_map.items() + missing_upstreams.items())
