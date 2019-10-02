# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

from google.protobuf import descriptor

SCALAR_TYPES = {
    descriptor.FieldDescriptor.TYPE_BOOL,
    descriptor.FieldDescriptor.TYPE_BYTES,
    descriptor.FieldDescriptor.TYPE_DOUBLE,
    descriptor.FieldDescriptor.TYPE_FIXED32,
    descriptor.FieldDescriptor.TYPE_FIXED64,
    descriptor.FieldDescriptor.TYPE_FLOAT,
    descriptor.FieldDescriptor.TYPE_INT32,
    descriptor.FieldDescriptor.TYPE_INT64,
    descriptor.FieldDescriptor.TYPE_SFIXED32,
    descriptor.FieldDescriptor.TYPE_SFIXED64,
    descriptor.FieldDescriptor.TYPE_SINT32,
    descriptor.FieldDescriptor.TYPE_SINT64,
    descriptor.FieldDescriptor.TYPE_STRING,
    descriptor.FieldDescriptor.TYPE_UINT32,
    descriptor.FieldDescriptor.TYPE_UINT64,
}


def merge_dict(data, msg):
  """Merges |data| dict into |msg|, recursively.

  Raises:
    TypeError if a field in |data| is not defined in |msg| or has an
    unsupported type.
  """
  if not isinstance(data, dict):
    raise TypeError('data is not a dict')
  for name, value in data.iteritems():
    f = msg.DESCRIPTOR.fields_by_name.get(name)
    if not f:
      raise TypeError('unexpected property %r' % name)
    scalar_f = f.type in SCALAR_TYPES
    msg_f = f.type == descriptor.FieldDescriptor.TYPE_MESSAGE
    if not scalar_f and not msg_f:  # pragma: no cover
      raise TypeError('field %s has unsupported type %r', name, f.type)

    try:
      if f.label == descriptor.FieldDescriptor.LABEL_REPEATED:
        container = getattr(msg, name)
        for v in value:
          if scalar_f:
            container.append(v)
          else:
            submsg = container.add()
            merge_dict(v, submsg)
      else:
        if scalar_f:
          setattr(msg, name, value)
        else:
          merge_dict(value, getattr(msg, name))
    except TypeError as ex:
      raise TypeError('%s: %s' % (name, ex))
