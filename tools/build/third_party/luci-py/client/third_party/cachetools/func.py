"""`functools.lru_cache` compatible memoizing function decorators."""

from __future__ import absolute_import

import collections
import functools
import random
import time

try:
    from threading import RLock
except ImportError:
    from dummy_threading import RLock

from . import keys
from .lfu import LFUCache
from .lru import LRUCache
from .rr import RRCache
from .ttl import TTLCache

__all__ = ('lfu_cache', 'lru_cache', 'rr_cache', 'ttl_cache')


_CacheInfo = collections.namedtuple('CacheInfo', [
    'hits', 'misses', 'maxsize', 'currsize'
])


def _cache(cache, typed=False):
    def decorator(func):
        key = keys.typedkey if typed else keys.hashkey
        lock = RLock()
        stats = [0, 0]

        def cache_info():
            with lock:
                hits, misses = stats
                maxsize = cache.maxsize
                currsize = cache.currsize
            return _CacheInfo(hits, misses, maxsize, currsize)

        def cache_clear():
            with lock:
                try:
                    cache.clear()
                finally:
                    stats[:] = [0, 0]

        def wrapper(*args, **kwargs):
            k = key(*args, **kwargs)
            with lock:
                try:
                    v = cache[k]
                    stats[0] += 1
                    return v
                except KeyError:
                    stats[1] += 1
            v = func(*args, **kwargs)
            try:
                with lock:
                    cache[k] = v
            except ValueError:
                pass  # value too large
            return v
        functools.update_wrapper(wrapper, func)
        if not hasattr(wrapper, '__wrapped__'):
            wrapper.__wrapped__ = func  # Python 2.7
        wrapper.cache_info = cache_info
        wrapper.cache_clear = cache_clear
        return wrapper
    return decorator


def lfu_cache(maxsize=128, typed=False):
    """Decorator to wrap a function with a memoizing callable that saves
    up to `maxsize` results based on a Least Frequently Used (LFU)
    algorithm.

    """
    return _cache(LFUCache(maxsize), typed)


def lru_cache(maxsize=128, typed=False):
    """Decorator to wrap a function with a memoizing callable that saves
    up to `maxsize` results based on a Least Recently Used (LRU)
    algorithm.

    """
    return _cache(LRUCache(maxsize), typed)


def rr_cache(maxsize=128, choice=random.choice, typed=False):
    """Decorator to wrap a function with a memoizing callable that saves
    up to `maxsize` results based on a Random Replacement (RR)
    algorithm.

    """
    return _cache(RRCache(maxsize, choice), typed)


def ttl_cache(maxsize=128, ttl=600, timer=time.time, typed=False):
    """Decorator to wrap a function with a memoizing callable that saves
    up to `maxsize` results based on a Least Recently Used (LRU)
    algorithm with a per-item time-to-live (TTL) value.
    """
    return _cache(TTLCache(maxsize, ttl, timer), typed)
