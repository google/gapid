# Copyright (C) 2010-2016  Vincent Pelletier <plr.vincent@gmail.com>
#
# This library is free software; you can redistribute it and/or
# modify it under the terms of the GNU Lesser General Public
# License as published by the Free Software Foundation; either
# version 2.1 of the License, or (at your option) any later version.
#
# This library is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
# Lesser General Public License for more details.
#
# You should have received a copy of the GNU Lesser General Public
# License along with this library; if not, write to the Free Software
# Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301  USA

# pylint: disable=invalid-name, too-few-public-methods, too-many-arguments
# pylint: disable=missing-docstring
"""
Python ctypes bindings for libusb-1.0.

You should not need to import this if you use usb1 module.

Declares all constants, data structures and exported symbols.
Locates and loads libusb1 dynamic library.
"""
from ctypes import Structure, LittleEndianStructure, \
    CFUNCTYPE, POINTER, addressof, sizeof, cast, \
    c_short, c_int, c_uint, c_size_t, c_long, \
    c_uint8, c_uint16, c_uint32, \
    c_void_p, c_char_p, py_object, string_at, pointer
try:
    from ctypes import c_ssize_t
except ImportError:
    from ctypes import c_longlong
    # c_ssize_t is new in Python 2.7
    if sizeof(c_int) == sizeof(c_size_t):
        c_ssize_t = c_int
    elif sizeof(c_long) == sizeof(c_size_t):
        c_ssize_t = c_long
    elif sizeof(c_longlong) == sizeof(c_size_t):
        c_ssize_t = c_longlong
    else:
        raise ValueError('Unsupported arch: sizeof(c_size_t) = %r' % (
            sizeof(c_size_t), ))
import ctypes.util
import platform
import os.path
import sys

class Enum(object):
    def __init__(self, member_dict, scope_dict=None):
        if scope_dict is None:
            # Affect caller's locals, not this module's.
            # pylint: disable=protected-access
            scope_dict = sys._getframe(1).f_locals
            # pylint: enable=protected-access
        forward_dict = {}
        reverse_dict = {}
        next_value = 0
        for name, value in member_dict.items():
            if value is None:
                value = next_value
                next_value += 1
            forward_dict[name] = value
            if value in reverse_dict:
                raise ValueError('Multiple names for value %r: %r, %r' % (
                    value, reverse_dict[value], name
                ))
            reverse_dict[value] = name
            scope_dict[name] = value
        self.forward_dict = forward_dict
        self.reverse_dict = reverse_dict

    def __call__(self, value):
        return self.reverse_dict[value]

    def get(self, value, default=None):
        return self.reverse_dict.get(value, default)

_desc_type_dict = {
    'b': c_uint8,
    'bcd': c_uint16,
    'bm': c_uint8,
    'dw': c_uint32,
    'i': c_uint8,
    'id': c_uint16,
    'w': c_uint16,
}

def newStruct(field_name_list):
    """
    Create a ctype structure class based on USB standard field naming
    (type-prefixed).
    """
    field_list = []
    append = field_list.append
    for field in field_name_list:
        type_prefix = ''
        for char in field:
            if not char.islower():
                break
            type_prefix += char
        append((field, _desc_type_dict[type_prefix]))
    result = type('some_descriptor', (LittleEndianStructure, ), {})
    # Not using type()'s 3rd param to initialise class, as per ctypes
    # documentation:
    #   _pack_ must already be defined when _fields_ is assigned, otherwise it
    #   will have no effect.
    # pylint: disable=protected-access
    result._pack_ = 1
    result._fields_ = field_list
    # pylint: enable=protected-access
    return result

def newDescriptor(field_name_list):
    """
    Create a USB descriptor ctype structure, ie starting with bLength and
    bDescriptorType fields.

    See newStruct().
    """
    return newStruct(['bLength', 'bDescriptorType'] + list(field_name_list))

class USBError(Exception):
    value = None

    def __init__(self, value=None):
        Exception.__init__(self)
        if value is not None:
            self.value = value

    def __str__(self):
        return '%s [%s]' % (libusb_error.get(self.value, 'Unknown error'),
                            self.value)

if sys.version_info[0] == 3:
    _string_item_to_int = lambda x: x
    _empty_char_p = bytes()
else:
    _string_item_to_int = ord
    _empty_char_p = ''

c_uchar = c_uint8
c_int_p = POINTER(c_int)

LITTLE_ENDIAN = sys.byteorder == 'little'

class timeval(Structure):
    _fields_ = [('tv_sec', c_long),
                ('tv_usec', c_long)]
timeval_p = POINTER(timeval)

def _loadLibrary():
    system = platform.system()
    if system == 'Windows':
        dll_loader = ctypes.WinDLL
        suffix = '.dll'
    else:
        dll_loader = ctypes.CDLL
        suffix = system == 'Darwin' and '.dylib' or '.so'
    loader_kw = {}
    if sys.version_info[:2] >= (2, 6):
        loader_kw['use_errno'] = True
        loader_kw['use_last_error'] = True
    try:
        return dll_loader('libusb-1.0' + suffix, **loader_kw)
    except OSError:
        libusb_path = None
        base_name = 'usb-1.0'
        if 'FreeBSD' in system:
            # libusb.so.2 on FreeBSD: load('libusb.so') would work fine, but...
            # libusb.so.2debian on Debian GNU/kFreeBSD: here it wouldn't work.
            # So use find_library instead.
            base_name = 'usb'
        elif system == 'Darwin':
            for libusb_path in (
                    # macport standard path
                    '/opt/local/lib/libusb-1.0.dylib',
                    # fink standard path
                    '/sw/lib/libusb-1.0.dylib',
                ):
                if os.path.exists(libusb_path):
                    break
            else:
                libusb_path = None
        if libusb_path is None:
            libusb_path = ctypes.util.find_library(base_name)
            if libusb_path is None:
                raise
        return dll_loader(libusb_path, **loader_kw)

libusb = _loadLibrary()

# libusb.h
def bswap16(x):
    return ((x & 0xff) << 8) | (x >> 8)

if LITTLE_ENDIAN:
    def libusb_cpu_to_le16(x):
        return x
    def libusb_le16_to_cpu(x):
        return x
else:
    libusb_cpu_to_le16 = bswap16
    libusb_le16_to_cpu = bswap16

# standard USB stuff

# Device and/or Interface Class codes
libusb_class_code = Enum({
    # In the context of a device descriptor,
    # this bDeviceClass value indicates that each interface specifies its
    # own class information and all interfaces operate independently.
    'LIBUSB_CLASS_PER_INTERFACE': 0,
    # Audio class
    'LIBUSB_CLASS_AUDIO': 1,
    # Communications class
    'LIBUSB_CLASS_COMM': 2,
    # Human Interface Device class
    'LIBUSB_CLASS_HID': 3,
    # Physical
    'LIBUSB_CLASS_PHYSICAL': 5,
    # Printer class
    'LIBUSB_CLASS_PRINTER': 7,
    # Picture transfer protocol class
    'LIBUSB_CLASS_PTP': 6,
    # Mass storage class
    'LIBUSB_CLASS_MASS_STORAGE': 8,
    # Hub class
    'LIBUSB_CLASS_HUB': 9,
    # Data class
    'LIBUSB_CLASS_DATA': 10,
    # Smart Card
    'LIBUSB_CLASS_SMART_CARD': 0x0b,
    # Content Security
    'LIBUSB_CLASS_CONTENT_SECURITY': 0x0d,
    # Video
    'LIBUSB_CLASS_VIDEO': 0x0e,
    # Personal Healthcare
    'LIBUSB_CLASS_PERSONAL_HEALTHCARE': 0x0f,
    # Diagnostic Device
    'LIBUSB_CLASS_DIAGNOSTIC_DEVICE': 0xdc,
    # Wireless class
    'LIBUSB_CLASS_WIRELESS': 0xe0,
    # Application class
    'LIBUSB_CLASS_APPLICATION': 0xfe,
    # Class is vendor-specific
    'LIBUSB_CLASS_VENDOR_SPEC': 0xff
})
# pylint: disable=undefined-variable
LIBUSB_CLASS_IMAGE = LIBUSB_CLASS_PTP
# pylint: enable=undefined-variable

# Descriptor types as defined by the USB specification.
libusb_descriptor_type = Enum({
    # Device descriptor. See libusb_device_descriptor.
    'LIBUSB_DT_DEVICE': 0x01,
    # Configuration descriptor. See libusb_config_descriptor.
    'LIBUSB_DT_CONFIG': 0x02,
    # String descriptor
    'LIBUSB_DT_STRING': 0x03,
    # Interface descriptor. See libusb_interface_descriptor.
    'LIBUSB_DT_INTERFACE': 0x04,
    # Endpoint descriptor. See libusb_endpoint_descriptor.
    'LIBUSB_DT_ENDPOINT': 0x05,
    # HID descriptor
    'LIBUSB_DT_HID': 0x21,
    # HID report descriptor
    'LIBUSB_DT_REPORT': 0x22,
    # Physical descriptor
    'LIBUSB_DT_PHYSICAL': 0x23,
    # Hub descriptor
    'LIBUSB_DT_HUB': 0x29,
})

# Descriptor sizes per descriptor type
LIBUSB_DT_DEVICE_SIZE = 18
LIBUSB_DT_CONFIG_SIZE = 9
LIBUSB_DT_INTERFACE_SIZE = 9
LIBUSB_DT_ENDPOINT_SIZE = 7
LIBUSB_DT_ENDPOINT_AUDIO_SIZE = 9 # Audio extension
LIBUSB_DT_HUB_NONVAR_SIZE = 7

LIBUSB_ENDPOINT_ADDRESS_MASK = 0x0f # in bEndpointAddress
LIBUSB_ENDPOINT_DIR_MASK = 0x80
# BBB
USB_ENDPOINT_ADDRESS_MASK = LIBUSB_ENDPOINT_ADDRESS_MASK
USB_ENDPOINT_DIR_MASK = LIBUSB_ENDPOINT_DIR_MASK

# Endpoint direction. Values for bit 7 of the endpoint address scheme.
libusb_endpoint_direction = Enum({
    # In: device-to-host
    'LIBUSB_ENDPOINT_IN': 0x80,
    # Out: host-to-device
    'LIBUSB_ENDPOINT_OUT': 0x00
})

LIBUSB_TRANSFER_TYPE_MASK = 0x03 # in bmAttributes

# Endpoint transfer type. Values for bits 0:1 of the endpoint attributes field.
libusb_transfer_type = Enum({
    # Control endpoint
    'LIBUSB_TRANSFER_TYPE_CONTROL': 0,
    # Isochronous endpoint
    'LIBUSB_TRANSFER_TYPE_ISOCHRONOUS': 1,
    # Bulk endpoint
    'LIBUSB_TRANSFER_TYPE_BULK': 2,
    # Interrupt endpoint
    'LIBUSB_TRANSFER_TYPE_INTERRUPT': 3,
})

# Standard requests, as defined in table 9-3 of the USB2 specifications
libusb_standard_request = Enum({
    # Request status of the specific recipient
    'LIBUSB_REQUEST_GET_STATUS': 0x00,
    # Clear or disable a specific feature
    'LIBUSB_REQUEST_CLEAR_FEATURE': 0x01,
    # 0x02 is reserved
    # Set or enable a specific feature
    'LIBUSB_REQUEST_SET_FEATURE': 0x03,
    # 0x04 is reserved
    # Set device address for all future accesses
    'LIBUSB_REQUEST_SET_ADDRESS': 0x05,
    # Get the specified descriptor
    'LIBUSB_REQUEST_GET_DESCRIPTOR': 0x06,
    # Used to update existing descriptors or add new descriptors
    'LIBUSB_REQUEST_SET_DESCRIPTOR': 0x07,
    # Get the current device configuration value
    'LIBUSB_REQUEST_GET_CONFIGURATION': 0x08,
    # Set device configuration
    'LIBUSB_REQUEST_SET_CONFIGURATION': 0x09,
    # Return the selected alternate setting for the specified interface
    'LIBUSB_REQUEST_GET_INTERFACE': 0x0a,
    # Select an alternate interface for the specified interface
    'LIBUSB_REQUEST_SET_INTERFACE': 0x0b,
    # Set then report an endpoint's synchronization frame
    'LIBUSB_REQUEST_SYNCH_FRAME': 0x0c,
})

# Request type bits of the bmRequestType field in control transfers.
libusb_request_type = Enum({
    # Standard
    'LIBUSB_REQUEST_TYPE_STANDARD': (0x00 << 5),
    # Class
    'LIBUSB_REQUEST_TYPE_CLASS': (0x01 << 5),
    # Vendor
    'LIBUSB_REQUEST_TYPE_VENDOR': (0x02 << 5),
    # Reserved
    'LIBUSB_REQUEST_TYPE_RESERVED': (0x03 << 5),
})

# BBB
# pylint: disable=bad-whitespace,undefined-variable
LIBUSB_TYPE_STANDARD = LIBUSB_REQUEST_TYPE_STANDARD
LIBUSB_TYPE_CLASS    = LIBUSB_REQUEST_TYPE_CLASS
LIBUSB_TYPE_VENDOR   = LIBUSB_REQUEST_TYPE_VENDOR
LIBUSB_TYPE_RESERVED = LIBUSB_REQUEST_TYPE_RESERVED
# pylint: enable=bad-whitespace,undefined-variable

# Recipient bits of the bmRequestType field in control transfers. Values 4
# through 31 are reserved.
libusb_request_recipient = Enum({
    # Device
    'LIBUSB_RECIPIENT_DEVICE': 0x00,
    # Interface
    'LIBUSB_RECIPIENT_INTERFACE': 0x01,
    # Endpoint
    'LIBUSB_RECIPIENT_ENDPOINT': 0x02,
    # Other
    'LIBUSB_RECIPIENT_OTHER': 0x03,
})

LIBUSB_ISO_SYNC_TYPE_MASK = 0x0c

# Synchronization type for isochronous endpoints. Values for bits 2:3 of the
# bmAttributes field in libusb_endpoint_descriptor.
libusb_iso_sync_type = Enum({
    # No synchronization
    'LIBUSB_ISO_SYNC_TYPE_NONE': 0,
    # Asynchronous
    'LIBUSB_ISO_SYNC_TYPE_ASYNC': 1,
    # Adaptive
    'LIBUSB_ISO_SYNC_TYPE_ADAPTIVE': 2,
    # Synchronous
    'LIBUSB_ISO_SYNC_TYPE_SYNC': 3,
})

LIBUSB_ISO_USAGE_TYPE_MASK = 0x30

# Usage type for isochronous endpoints. Values for bits 4:5 of the
# bmAttributes field in libusb_endpoint_descriptor.
libusb_iso_usage_type = Enum({
    # Data endpoint
    'LIBUSB_ISO_USAGE_TYPE_DATA': 0,
    # Feedback endpoint
    'LIBUSB_ISO_USAGE_TYPE_FEEDBACK': 1,
    # Implicit feedback Data endpoint
    'LIBUSB_ISO_USAGE_TYPE_IMPLICIT': 2,
})

# A structure representing the standard USB device descriptor. This
# descriptor is documented in section 9.6.1 of the USB 2.0 specification.
# All multiple-byte fields are represented in host-endian format.
class libusb_device_descriptor(Structure):
    _fields_ = [
        # Size of this descriptor (in bytes)
        ('bLength', c_uint8),
        # Descriptor type. Will have value LIBUSB_DT_DEVICE in this
        # context.
        ('bDescriptorType', c_uint8),
        # USB specification release number in binary-coded decimal. A
        # value of 0x0200 indicates USB 2.0, 0x0110 indicates USB 1.1,
        # etc.
        ('bcdUSB', c_uint16),
        # USB-IF class code for the device. See libusb_class_code.
        ('bDeviceClass', c_uint8),
        # USB-IF subclass code for the device, qualified by the
        # bDeviceClass value
        ('bDeviceSubClass', c_uint8),
        # USB-IF protocol code for the device, qualified by the
        # bDeviceClass and bDeviceSubClass values
        ('bDeviceProtocol', c_uint8),
        # Maximum packet size for endpoint 0
        ('bMaxPacketSize0', c_uint8),
        # USB-IF vendor ID
        ('idVendor', c_uint16),
        # USB-IF product ID
        ('idProduct', c_uint16),
        # Device release number in binary-coded decimal
        ('bcdDevice', c_uint16),
        # Index of string descriptor describing manufacturer
        ('iManufacturer', c_uint8),
        # Index of string descriptor describing product
        ('iProduct', c_uint8),
        # Index of string descriptor containing device serial number
        ('iSerialNumber', c_uint8),
        # Number of possible configurations
        ('bNumConfigurations', c_uint8)]
libusb_device_descriptor_p = POINTER(libusb_device_descriptor)

class libusb_endpoint_descriptor(Structure):
    _fields_ = [
        ('bLength', c_uint8),
        ('bDescriptorType', c_uint8),
        ('bEndpointAddress', c_uint8),
        ('bmAttributes', c_uint8),
        ('wMaxPacketSize', c_uint16),
        ('bInterval', c_uint8),
        ('bRefresh', c_uint8),
        ('bSynchAddress', c_uint8),
        ('extra', c_void_p),
        ('extra_length', c_int)]
libusb_endpoint_descriptor_p = POINTER(libusb_endpoint_descriptor)

class libusb_interface_descriptor(Structure):
    _fields_ = [
        ('bLength', c_uint8),
        ('bDescriptorType', c_uint8),
        ('bInterfaceNumber', c_uint8),
        ('bAlternateSetting', c_uint8),
        ('bNumEndpoints', c_uint8),
        ('bInterfaceClass', c_uint8),
        ('bInterfaceSubClass', c_uint8),
        ('bInterfaceProtocol', c_uint8),
        ('iInterface', c_uint8),
        ('endpoint', libusb_endpoint_descriptor_p),
        ('extra', c_void_p),
        ('extra_length', c_int)]
libusb_interface_descriptor_p = POINTER(libusb_interface_descriptor)

class libusb_interface(Structure):
    _fields_ = [('altsetting', libusb_interface_descriptor_p),
                ('num_altsetting', c_int)]
libusb_interface_p = POINTER(libusb_interface)

class libusb_config_descriptor(Structure):
    _fields_ = [
        ('bLength', c_uint8),
        ('bDescriptorType', c_uint8),
        ('wTotalLength', c_uint16),
        ('bNumInterfaces', c_uint8),
        ('bConfigurationValue', c_uint8),
        ('iConfiguration', c_uint8),
        ('bmAttributes', c_uint8),
        ('MaxPower', c_uint8),
        ('interface', libusb_interface_p),
        ('extra', c_void_p),
        ('extra_length', c_int)]
libusb_config_descriptor_p = POINTER(libusb_config_descriptor)
libusb_config_descriptor_p_p = POINTER(libusb_config_descriptor_p)

class libusb_control_setup(Structure):
    _fields_ = [
        ('bmRequestType', c_uint8),
        ('bRequest', c_uint8),
        ('wValue', c_uint16),
        ('wIndex', c_uint16),
        ('wLength', c_uint16)]
libusb_control_setup_p = POINTER(libusb_control_setup)

LIBUSB_CONTROL_SETUP_SIZE = sizeof(libusb_control_setup)

# Structure representing a libusb session. The concept of individual libusb
# sessions allows for your program to use two libraries (or dynamically
# load two modules) which both independently use libusb. This will prevent
# interference between the individual libusb users - for example
# libusb_set_debug() will not affect the other user of the library, and
# libusb_exit() will not destroy resources that the other user is still
# using.
#
# Sessions are created by libusb_init() and destroyed through libusb_exit().
# If your application is guaranteed to only ever include a single libusb
# user (i.e. you), you do not have to worry about contexts: pass NULL in
# every function call where a context is required. The default context
# will be used.
#
# For more information, see \ref contexts.
class libusb_context(Structure):
    pass
libusb_context_p = POINTER(libusb_context)
libusb_context_p_p = POINTER(libusb_context_p)

# Structure representing a USB device detected on the system. This is an
# opaque type for which you are only ever provided with a pointer, usually
# originating from libusb_get_device_list().
#
# Certain operations can be performed on a device, but in order to do any
# I/O you will have to first obtain a device handle using libusb_open().
#
# Devices are reference counted with libusb_device_ref() and
# libusb_device_unref(), and are freed when the reference count reaches 0.
# New devices presented by libusb_get_device_list() have a reference count of
# 1, and libusb_free_device_list() can optionally decrease the reference count
# on all devices in the list. libusb_open() adds another reference which is
# later destroyed by libusb_close().
class libusb_device(Structure):
    pass
libusb_device_p = POINTER(libusb_device)
libusb_device_p_p = POINTER(libusb_device_p)
libusb_device_p_p_p = POINTER(libusb_device_p_p)

# Structure representing a handle on a USB device. This is an opaque type for
# which you are only ever provided with a pointer, usually originating from
# libusb_open().
#
# A device handle is used to perform I/O and other operations. When finished
# with a device handle, you should call libusb_close().
class libusb_device_handle(Structure):
    pass
libusb_device_handle_p = POINTER(libusb_device_handle)
libusb_device_handle_p_p = POINTER(libusb_device_handle_p)

class libusb_version(Structure):
    _fields_ = [
        ('major', c_uint16),
        ('minor', c_uint16),
        ('micro', c_uint16),
        ('nano', c_uint16),
        ('rc', c_char_p),
        ('describe', c_char_p),
    ]

libusb_speed = Enum({
    # The OS doesn't report or know the device speed.
    'LIBUSB_SPEED_UNKNOWN': 0,
    # The device is operating at low speed (1.5MBit/s).
    'LIBUSB_SPEED_LOW': 1,
    # The device is operating at full speed (12MBit/s).
    'LIBUSB_SPEED_FULL': 2,
    # The device is operating at high speed (480MBit/s).
    'LIBUSB_SPEED_HIGH': 3,
    # The device is operating at super speed (5000MBit/s).
    'LIBUSB_SPEED_SUPER': 4,
})

libusb_supported_speed = Enum({
    # Low speed operation supported (1.5MBit/s).
    'LIBUSB_LOW_SPEED_OPERATION': 1,
    # Full speed operation supported (12MBit/s).
    'LIBUSB_FULL_SPEED_OPERATION': 2,
    # High speed operation supported (480MBit/s).
    'LIBUSB_HIGH_SPEED_OPERATION': 4,
    # Superspeed operation supported (5000MBit/s).
    'LIBUSB_5GBPS_OPERATION': 8,
})

# Error codes. Most libusb functions return 0 on success or one of these
# codes on failure.
libusb_error = Enum({
    # Success (no error)
    'LIBUSB_SUCCESS': 0,
    # Input/output error
    'LIBUSB_ERROR_IO': -1,
    # Invalid parameter
    'LIBUSB_ERROR_INVALID_PARAM': -2,
    # Access denied (insufficient permissions)
    'LIBUSB_ERROR_ACCESS': -3,
    # No such device (it may have been disconnected)
    'LIBUSB_ERROR_NO_DEVICE': -4,
    # Entity not found
    'LIBUSB_ERROR_NOT_FOUND': -5,
    # Resource busy
    'LIBUSB_ERROR_BUSY': -6,
    # Operation timed out
    'LIBUSB_ERROR_TIMEOUT': -7,
    # Overflow
    'LIBUSB_ERROR_OVERFLOW': -8,
    # Pipe error
    'LIBUSB_ERROR_PIPE': -9,
    # System call interrupted (perhaps due to signal)
    'LIBUSB_ERROR_INTERRUPTED': -10,
    # Insufficient memory
    'LIBUSB_ERROR_NO_MEM': -11,
    # Operation not supported or unimplemented on this platform
    'LIBUSB_ERROR_NOT_SUPPORTED': -12,
    # Other error
    'LIBUSB_ERROR_OTHER': -99,
})

# Transfer status codes
libusb_transfer_status = Enum({
    # Transfer completed without error. Note that this does not indicate
    # that the entire amount of requested data was transferred.
    'LIBUSB_TRANSFER_COMPLETED': 0,
    # Transfer failed
    'LIBUSB_TRANSFER_ERROR': 1,
    # Transfer timed out
    'LIBUSB_TRANSFER_TIMED_OUT': 2,
    # Transfer was cancelled
    'LIBUSB_TRANSFER_CANCELLED': 3,
    # For bulk/interrupt endpoints: halt condition detected (endpoint
    # stalled). For control endpoints: control request not supported.
    'LIBUSB_TRANSFER_STALL': 4,
    # Device was disconnected
    'LIBUSB_TRANSFER_NO_DEVICE': 5,
    # Device sent more data than requested
    'LIBUSB_TRANSFER_OVERFLOW': 6,
})

# libusb_transfer.flags values
libusb_transfer_flags = Enum({
    # Report short frames as errors
    'LIBUSB_TRANSFER_SHORT_NOT_OK': 1 << 0,
    # Automatically free() transfer buffer during libusb_free_transfer()
    'LIBUSB_TRANSFER_FREE_BUFFER': 1 << 1,
    # Automatically call libusb_free_transfer() after callback returns.
    # If this flag is set, it is illegal to call libusb_free_transfer()
    # from your transfer callback, as this will result in a double-free
    # when this flag is acted upon.
    'LIBUSB_TRANSFER_FREE_TRANSFER': 1 << 2,
    # Terminate transfers that are a multiple of the endpoint's
    # wMaxPacketSize with an extra zero length packet.
    'LIBUSB_TRANSFER_ADD_ZERO_PACKET': 1 << 3,
})

# Isochronous packet descriptor.
class libusb_iso_packet_descriptor(Structure):
    _fields_ = [('length', c_uint),
                ('actual_length', c_uint),
                ('status', c_int)] # enum libusb_transfer_status
libusb_iso_packet_descriptor_p = POINTER(libusb_iso_packet_descriptor)

class libusb_transfer(Structure):
    pass
libusb_transfer_p = POINTER(libusb_transfer)

libusb_transfer_cb_fn_p = CFUNCTYPE(None, libusb_transfer_p)

_libusb_transfer_fields = [
    ('dev_handle', libusb_device_handle_p),
    ('flags', c_uint8),
    ('endpoint', c_uchar),
    ('type', c_uchar),
    ('timeout', c_uint),
    ('status', c_int), # enum libusb_transfer_status
    ('length', c_int),
    ('actual_length', c_int),
    ('callback', libusb_transfer_cb_fn_p),
    ('user_data', c_void_p),
    ('buffer', c_void_p),
    ('num_iso_packets', c_int),
    ('iso_packet_desc', libusb_iso_packet_descriptor)
]
if 'FreeBSD' in platform.system() and getattr(
        libusb, 'libusb_get_string_descriptor', None
    ) is None:
    # Old FreeBSD version has a slight ABI incompatibility.
    # Work around it unless libusb_get_string_descriptor is available, as it
    # is only available on fixed versions.
    assert _libusb_transfer_fields[2][0] == 'endpoint'
    _libusb_transfer_fields[2] = ('endpoint', c_uint32)
    assert _libusb_transfer_fields[11][0] == 'num_iso_packets'
    _libusb_transfer_fields.insert(11, ('os_priv', c_void_p))

# pylint: disable=protected-access
libusb_transfer._fields_ = _libusb_transfer_fields
# pylint: enable=protected-access

libusb_capability = Enum({
    # The libusb_has_capability() API is available.
    'LIBUSB_CAP_HAS_CAPABILITY': 0x0000,
    # Hotplug support is available.
    'LIBUSB_CAP_HAS_HOTPLUG': 0x0001,
    # The library can access HID devices without requiring user intervention.
    'LIBUSB_CAP_HAS_HID_ACCESS': 0x0100,
    # The library supports detaching of the default USB driver.
    'LIBUSB_CAP_SUPPORTS_DETACH_KERNEL_DRIVER': 0x0101,
})

libusb_log_level = Enum({
    'LIBUSB_LOG_LEVEL_NONE': 0,
    'LIBUSB_LOG_LEVEL_ERROR': 1,
    'LIBUSB_LOG_LEVEL_WARNING': 2,
    'LIBUSB_LOG_LEVEL_INFO': 3,
    'LIBUSB_LOG_LEVEL_DEBUG': 4,
})

#int libusb_init(libusb_context **ctx);
libusb_init = libusb.libusb_init
libusb_init.argtypes = [libusb_context_p_p]
#void libusb_exit(libusb_context *ctx);
libusb_exit = libusb.libusb_exit
libusb_exit.argtypes = [libusb_context_p]
libusb_exit.restype = None
#void libusb_set_debug(libusb_context *ctx, int level);
libusb_set_debug = libusb.libusb_set_debug
libusb_set_debug.argtypes = [libusb_context_p, c_int]
libusb_set_debug.restype = None
#const struct libusb_version * libusb_get_version(void);
try:
    libusb_get_version = libusb.libusb_get_version
except AttributeError:
    _dummy_version = libusb_version(0, 0, 0, 0, _empty_char_p, _empty_char_p)
    _dummy_version_p = pointer(_dummy_version)
    def libusb_get_version():
        return _dummy_version_p
else:
    libusb_get_version.argtypes = []
    libusb_get_version.restype = POINTER(libusb_version)
#int libusb_has_capability(uint32_t capability);
try:
    libusb_has_capability = libusb.libusb_has_capability
except AttributeError:
    def libusb_has_capability(_):
        return 0
else:
    libusb_has_capability.argtypes = [c_uint32]
    libusb_has_capability.restype = c_int
try:
    # Note: Should be equivalent to libusb_error.get (except libusb_error.get
    # one raises on unknown values).
    #char *libusb_error_name(int errcode);
    libusb_error_name = libusb.libusb_error_name
except AttributeError:
    # pylint: disable=unused-argument
    def libusb_error_name(errcode):
        return None
    # pylint: enable=unused-argument
else:
    libusb_error_name.argtypes = [c_int]
    libusb_error_name.restype = c_char_p

# Note on libusb_strerror, libusb_setlocale and future functions in the
# same spirit:
# I do not think end-user-facing messages belong to a technical library.
# Such features bring a new, non essential set of problems, and is a luxury
# I do not want to spend time supporting considering limited resources and
# more important stuff to work on.
# For backward compatibility, expose libusb_strerror placeholder.
# pylint: disable=unused-argument
def libusb_strerror(errcode):
    return None
# pylint: enable=unused-argument

#ssize_t libusb_get_device_list(libusb_context *ctx,
#        libusb_device ***list);
libusb_get_device_list = libusb.libusb_get_device_list
libusb_get_device_list.argtypes = [libusb_context_p, libusb_device_p_p_p]
libusb_get_device_list.restype = c_ssize_t
#void libusb_free_device_list(libusb_device **list, int unref_devices);
libusb_free_device_list = libusb.libusb_free_device_list
libusb_free_device_list.argtypes = [libusb_device_p_p, c_int]
libusb_free_device_list.restype = None
#libusb_device *libusb_ref_device(libusb_device *dev);
libusb_ref_device = libusb.libusb_ref_device
libusb_ref_device.argtypes = [libusb_device_p]
libusb_ref_device.restype = libusb_device_p
#void libusb_unref_device(libusb_device *dev);
libusb_unref_device = libusb.libusb_unref_device
libusb_unref_device.argtypes = [libusb_device_p]
libusb_unref_device.restype = None

#int libusb_get_configuration(libusb_device_handle *dev, int *config);
libusb_get_configuration = libusb.libusb_get_configuration
libusb_get_configuration.argtypes = [libusb_device_handle_p, c_int_p]
#int libusb_get_device_descriptor(libusb_device *dev,
#        struct libusb_device_descriptor *desc);
libusb_get_device_descriptor = libusb.libusb_get_device_descriptor
libusb_get_device_descriptor.argtypes = [
    libusb_device_p, libusb_device_descriptor_p]
#int libusb_get_active_config_descriptor(libusb_device *dev,
#        struct libusb_config_descriptor **config);
libusb_get_active_config_descriptor = libusb.libusb_get_active_config_descriptor
libusb_get_active_config_descriptor.argtypes = [
    libusb_device_p, libusb_config_descriptor_p_p]
#int libusb_get_config_descriptor(libusb_device *dev, uint8_t config_index,
#        struct libusb_config_descriptor **config);
libusb_get_config_descriptor = libusb.libusb_get_config_descriptor
libusb_get_config_descriptor.argtypes = [
    libusb_device_p, c_uint8, libusb_config_descriptor_p_p]
#int libusb_get_config_descriptor_by_value(libusb_device *dev,
#        uint8_t bConfigurationValue, struct libusb_config_descriptor **config);
libusb_get_config_descriptor_by_value = \
    libusb.libusb_get_config_descriptor_by_value
libusb_get_config_descriptor_by_value.argtypes = [
    libusb_device_p, c_uint8, libusb_config_descriptor_p_p]
#void libusb_free_config_descriptor(struct libusb_config_descriptor *config);
libusb_free_config_descriptor = libusb.libusb_free_config_descriptor
libusb_free_config_descriptor.argtypes = [libusb_config_descriptor_p]
libusb_free_config_descriptor.restype = None
#uint8_t libusb_get_bus_number(libusb_device *dev);
libusb_get_bus_number = libusb.libusb_get_bus_number
libusb_get_bus_number.argtypes = [libusb_device_p]
libusb_get_bus_number.restype = c_uint8
try:
    #uint8_t libusb_get_port_number(libusb_device *dev);
    libusb_get_port_number = libusb.libusb_get_port_number
except AttributeError:
    pass
else:
    libusb_get_port_number.argtypes = [libusb_device_p]
    libusb_get_port_number.restype = c_uint8
try:
    #int libusb_get_port_numbers(libusb_device *dev,
    #       uint8_t* port_numbers, int port_numbers_len);
    libusb_get_port_numbers = libusb.libusb_get_port_numbers
except AttributeError:
    pass
else:
    libusb_get_port_numbers.argtypes = [
        libusb_device_p, POINTER(c_uint8), c_int]
    libusb_get_port_numbers.restype = c_int
# Missing: libusb_get_port_path (deprecated since 1.0.16)
try:
    #libusb_device * LIBUSB_CALL libusb_get_parent(libusb_device *dev);
    libusb_get_parent = libusb.libusb_get_parent
except AttributeError:
    pass
else:
    libusb_get_parent.argtypes = [libusb_device_p]
    libusb_get_parent.restype = libusb_device_p
#uint8_t libusb_get_device_address(libusb_device *dev);
libusb_get_device_address = libusb.libusb_get_device_address
libusb_get_device_address.argtypes = [libusb_device_p]
libusb_get_device_address.restype = c_uint8
try:
    #int libusb_get_device_speed(libusb_device *dev);
    libusb_get_device_speed = libusb.libusb_get_device_speed
except AttributeError:
    # Place holder
    def libusb_get_device_speed(_):
        # pylint: disable=undefined-variable
        return LIBUSB_SPEED_UNKNOWN
        # pylint: enable=undefined-variable
else:
    libusb_get_device_speed.argtypes = [libusb_device_p]
#int libusb_get_max_packet_size(libusb_device *dev, unsigned char endpoint);
libusb_get_max_packet_size = libusb.libusb_get_max_packet_size
libusb_get_max_packet_size.argtypes = [libusb_device_p, c_uchar]
#int libusb_get_max_iso_packet_size(libusb_device *dev, unsigned char endpoint);
try:
    libusb_get_max_iso_packet_size = libusb.libusb_get_max_iso_packet_size
except AttributeError:
    # FreeBSD's reimplementation of the API [used to ]lack[s] this function.
    # It has been added in r234193, but is lacking in default 9.x install as
    # of this change. Provide a fallback to error-out only if actually used.
    # pylint: disable=unused-argument
    def libusb_get_max_iso_packet_size(_, __):
        raise NotImplementedError
    # pylint: enable=unused-argument
else:
    libusb_get_max_iso_packet_size.argtypes = [libusb_device_p, c_uchar]

#int libusb_open(libusb_device *dev, libusb_device_handle **handle);
libusb_open = libusb.libusb_open
libusb_open.argtypes = [libusb_device_p, libusb_device_handle_p_p]
#void libusb_close(libusb_device_handle *dev_handle);
libusb_close = libusb.libusb_close
libusb_close.argtypes = [libusb_device_handle_p]
libusb_close.restype = None
#libusb_device *libusb_get_device(libusb_device_handle *dev_handle);
libusb_get_device = libusb.libusb_get_device
libusb_get_device.argtypes = [libusb_device_handle_p]
libusb_get_device.restype = libusb_device_p

#int libusb_set_configuration(libusb_device_handle *dev, int configuration);
libusb_set_configuration = libusb.libusb_set_configuration
libusb_set_configuration.argtypes = [libusb_device_handle_p, c_int]
#int libusb_claim_interface(libusb_device_handle *dev, int iface);
libusb_claim_interface = libusb.libusb_claim_interface
libusb_claim_interface.argtypes = [libusb_device_handle_p, c_int]
#int libusb_release_interface(libusb_device_handle *dev, int iface);
libusb_release_interface = libusb.libusb_release_interface
libusb_release_interface.argtypes = [libusb_device_handle_p, c_int]

#libusb_device_handle *libusb_open_device_with_vid_pid(libusb_context *ctx,
#        uint16_t vendor_id, uint16_t product_id);
libusb_open_device_with_vid_pid = libusb.libusb_open_device_with_vid_pid
libusb_open_device_with_vid_pid.argtypes = [
    libusb_context_p, c_uint16, c_uint16]
libusb_open_device_with_vid_pid.restype = libusb_device_handle_p

#int libusb_set_interface_alt_setting(libusb_device_handle *dev,
#        int interface_number, int alternate_setting);
libusb_set_interface_alt_setting = libusb.libusb_set_interface_alt_setting
libusb_set_interface_alt_setting.argtypes = [
    libusb_device_handle_p, c_int, c_int]
#int libusb_clear_halt(libusb_device_handle *dev, unsigned char endpoint);
libusb_clear_halt = libusb.libusb_clear_halt
libusb_clear_halt.argtypes = [libusb_device_handle_p, c_uchar]
#int libusb_reset_device(libusb_device_handle *dev);
libusb_reset_device = libusb.libusb_reset_device
libusb_reset_device.argtypes = [libusb_device_handle_p]

#int libusb_kernel_driver_active(libusb_device_handle *dev, int interface);
libusb_kernel_driver_active = libusb.libusb_kernel_driver_active
libusb_kernel_driver_active.argtypes = [libusb_device_handle_p, c_int]
#int libusb_detach_kernel_driver(libusb_device_handle *dev, int interface);
libusb_detach_kernel_driver = libusb.libusb_detach_kernel_driver
libusb_detach_kernel_driver.argtypes = [libusb_device_handle_p, c_int]
#int libusb_attach_kernel_driver(libusb_device_handle *dev, int interface);
libusb_attach_kernel_driver = libusb.libusb_attach_kernel_driver
libusb_attach_kernel_driver.argtypes = [libusb_device_handle_p, c_int]
try:
    #int libusb_set_auto_detach_kernel_driver(
    #       libusb_device_handle *dev, int enable);
    libusb_set_auto_detach_kernel_driver = \
        libusb.libusb_set_auto_detach_kernel_driver
except AttributeError:
    pass
else:
    libusb_set_auto_detach_kernel_driver.argtypes = [
        libusb_device_handle_p, c_int]
    libusb_set_auto_detach_kernel_driver.restype = c_int

# Get the data section of a control transfer. This convenience function is here
# to remind you that the data does not start until 8 bytes into the actual
# buffer, as the setup packet comes first.
#
# Calling this function only makes sense from a transfer callback function,
# or situations where you have already allocated a suitably sized buffer at
# transfer->buffer.
#
# \param transfer a transfer
# \returns pointer to the first byte of the data section

def libusb_control_transfer_get_data(transfer_p):
    transfer = transfer_p.contents
    return string_at(transfer.buffer, transfer.length)[
        LIBUSB_CONTROL_SETUP_SIZE:]

def libusb_control_transfer_get_setup(transfer_p):
    return cast(transfer_p.contents.buffer, libusb_control_setup_p)

def libusb_fill_control_setup(
        setup_p, bmRequestType, bRequest, wValue, wIndex, wLength):
    setup = cast(setup_p, libusb_control_setup_p).contents
    setup.bmRequestType = bmRequestType
    setup.bRequest = bRequest
    setup.wValue = libusb_cpu_to_le16(wValue)
    setup.wIndex = libusb_cpu_to_le16(wIndex)
    setup.wLength = libusb_cpu_to_le16(wLength)

#struct libusb_transfer *libusb_alloc_transfer(int iso_packets);
libusb_alloc_transfer = libusb.libusb_alloc_transfer
libusb_alloc_transfer.argtypes = [c_int]
libusb_alloc_transfer.restype = libusb_transfer_p
#int libusb_submit_transfer(struct libusb_transfer *transfer);
libusb_submit_transfer = libusb.libusb_submit_transfer
libusb_submit_transfer.argtypes = [libusb_transfer_p]
#int libusb_cancel_transfer(struct libusb_transfer *transfer);
libusb_cancel_transfer = libusb.libusb_cancel_transfer
libusb_cancel_transfer.argtypes = [libusb_transfer_p]
#void libusb_free_transfer(struct libusb_transfer *transfer);
libusb_free_transfer = libusb.libusb_free_transfer
libusb_free_transfer.argtypes = [libusb_transfer_p]
libusb_free_transfer.restype = None

# pylint: disable=redefined-builtin
def libusb_fill_control_transfer(
        transfer_p, dev_handle, buffer, callback, user_data, timeout):
    transfer = transfer_p.contents
    transfer.dev_handle = dev_handle
    transfer.endpoint = 0
    # pylint: disable=undefined-variable
    transfer.type = LIBUSB_TRANSFER_TYPE_CONTROL
    # pylint: enable=undefined-variable
    transfer.timeout = timeout
    transfer.buffer = cast(buffer, c_void_p)
    if buffer is not None:
        setup = cast(buffer, libusb_control_setup_p).contents
        # pylint: disable=undefined-variable
        transfer.length = LIBUSB_CONTROL_SETUP_SIZE + \
            libusb_le16_to_cpu(setup.wLength)
        # pylint: enable=undefined-variable
    transfer.user_data = user_data
    transfer.callback = callback
# pylint: enable=redefined-builtin

# pylint: disable=redefined-builtin
def libusb_fill_bulk_transfer(
        transfer_p, dev_handle, endpoint, buffer, length,
        callback, user_data, timeout):
    transfer = transfer_p.contents
    transfer.dev_handle = dev_handle
    transfer.endpoint = endpoint
    # pylint: disable=undefined-variable
    transfer.type = LIBUSB_TRANSFER_TYPE_BULK
    # pylint: enable=undefined-variable
    transfer.timeout = timeout
    transfer.buffer = cast(buffer, c_void_p)
    transfer.length = length
    transfer.user_data = user_data
    transfer.callback = callback
# pylint: enable=redefined-builtin

# pylint: disable=redefined-builtin
def libusb_fill_interrupt_transfer(
        transfer_p, dev_handle, endpoint, buffer,
        length, callback, user_data, timeout):
    transfer = transfer_p.contents
    transfer.dev_handle = dev_handle
    transfer.endpoint = endpoint
    # pylint: disable=undefined-variable
    transfer.type = LIBUSB_TRANSFER_TYPE_INTERRUPT
    # pylint: enable=undefined-variable
    transfer.timeout = timeout
    transfer.buffer = cast(buffer, c_void_p)
    transfer.length = length
    transfer.user_data = user_data
    transfer.callback = callback
# pylint: enable=redefined-builtin

# pylint: disable=redefined-builtin
def libusb_fill_iso_transfer(
        transfer_p, dev_handle, endpoint, buffer, length,
        num_iso_packets, callback, user_data, timeout):
    transfer = transfer_p.contents
    transfer.dev_handle = dev_handle
    transfer.endpoint = endpoint
    # pylint: disable=undefined-variable
    transfer.type = LIBUSB_TRANSFER_TYPE_ISOCHRONOUS
    # pylint: enable=undefined-variable
    transfer.timeout = timeout
    transfer.buffer = cast(buffer, c_void_p)
    transfer.length = length
    transfer.num_iso_packets = num_iso_packets
    transfer.user_data = user_data
    transfer.callback = callback
# pylint: enable=redefined-builtin

def _get_iso_packet_list(transfer):
    list_type = libusb_iso_packet_descriptor * transfer.num_iso_packets
    return list_type.from_address(addressof(transfer.iso_packet_desc))

def get_iso_packet_list(transfer_p):
    """
    Python-specific helper extracting a list of iso packet descriptors,
    because it's not as straight-forward as in C.
    """
    return _get_iso_packet_list(transfer_p.contents)

def _get_iso_packet_buffer(transfer, offset, length):
    return string_at(transfer.buffer + offset, length)

def get_iso_packet_buffer_list(transfer_p):
    """
    Python-specific helper extracting a list of iso packet buffers.
    """
    transfer = transfer_p.contents
    offset = 0
    result = []
    append = result.append
    for iso_transfer in _get_iso_packet_list(transfer):
        length = iso_transfer.length
        append(_get_iso_packet_buffer(transfer, offset, length))
        offset += length
    return result

def get_extra(descriptor):
    """
    Python-specific helper to access "extra" field of descriptors,
    because it's not as straight-forward as in C.
    Returns a list, where each entry is an individual extra descriptor.
    """
    result = []
    extra_length = descriptor.extra_length
    if extra_length:
        extra = string_at(descriptor.extra, extra_length)
        append = result.append
        while extra:
            length = _string_item_to_int(extra[0])
            if not 0 < length <= len(extra):
                raise ValueError(
                    'Extra descriptor %i is incomplete/invalid' % (
                        len(result),
                    ),
                )
            append(extra[:length])
            extra = extra[length:]
    return result

def libusb_set_iso_packet_lengths(transfer_p, length):
    transfer = transfer_p.contents
    for iso_packet_desc in _get_iso_packet_list(transfer):
        iso_packet_desc.length = length

def libusb_get_iso_packet_buffer(transfer_p, packet):
    transfer = transfer_p.contents
    offset = 0
    if packet >= transfer.num_iso_packets:
        return None
    iso_packet_desc_list = _get_iso_packet_list(transfer)
    for i in xrange(packet):
        offset += iso_packet_desc_list[i].length
    return _get_iso_packet_buffer(
        transfer, offset, iso_packet_desc_list[packet].length)

def libusb_get_iso_packet_buffer_simple(transfer_p, packet):
    transfer = transfer_p.contents
    if packet >= transfer.num_iso_packets:
        return None
    iso_length = transfer.iso_packet_desc.length
    return _get_iso_packet_buffer(transfer, iso_length * packet, iso_length)

# sync I/O

#int libusb_control_transfer(libusb_device_handle *dev_handle,
#        uint8_t request_type, uint8_t request, uint16_t value, uint16_t index,
#        unsigned char *data, uint16_t length, unsigned int timeout);
libusb_control_transfer = libusb.libusb_control_transfer
libusb_control_transfer.argtypes = [libusb_device_handle_p, c_uint8, c_uint8,
                                    c_uint16, c_uint16, c_void_p, c_uint16,
                                    c_uint]

#int libusb_bulk_transfer(libusb_device_handle *dev_handle,
#        unsigned char endpoint, unsigned char *data, int length,
#        int *actual_length, unsigned int timeout);
libusb_bulk_transfer = libusb.libusb_bulk_transfer
libusb_bulk_transfer.argtypes = [libusb_device_handle_p, c_uchar, c_void_p,
                                 c_int, c_int_p, c_uint]

#int libusb_interrupt_transfer(libusb_device_handle *dev_handle,
#        unsigned char endpoint, unsigned char *data, int length,
#        int *actual_length, unsigned int timeout);
libusb_interrupt_transfer = libusb.libusb_interrupt_transfer
libusb_interrupt_transfer.argtypes = [libusb_device_handle_p, c_uchar,
                                      c_void_p, c_int, c_int_p, c_uint]

# pylint: disable=undefined-variable
def libusb_get_descriptor(dev, desc_type, desc_index, data, length):
    return libusb_control_transfer(dev, LIBUSB_ENDPOINT_IN,
                                   LIBUSB_REQUEST_GET_DESCRIPTOR,
                                   (desc_type << 8) | desc_index, 0, data,
                                   length, 1000)
# pylint: enable=undefined-variable

# pylint: disable=undefined-variable
def libusb_get_string_descriptor(dev, desc_index, langid, data, length):
    return libusb_control_transfer(dev, LIBUSB_ENDPOINT_IN,
                                   LIBUSB_REQUEST_GET_DESCRIPTOR,
                                   (LIBUSB_DT_STRING << 8) | desc_index,
                                   langid, data, length, 1000)
# pylint: enable=undefined-variable

#int libusb_get_string_descriptor_ascii(libusb_device_handle *dev,
#        uint8_t index, unsigned char *data, int length);
libusb_get_string_descriptor_ascii = libusb.libusb_get_string_descriptor_ascii
libusb_get_string_descriptor_ascii.argtypes = [libusb_device_handle_p,
                                               c_uint8, c_void_p, c_int]

# polling and timeouts

#int libusb_try_lock_events(libusb_context *ctx);
libusb_try_lock_events = libusb.libusb_try_lock_events
libusb_try_lock_events.argtypes = [libusb_context_p]
#void libusb_lock_events(libusb_context *ctx);
libusb_lock_events = libusb.libusb_lock_events
libusb_lock_events.argtypes = [libusb_context_p]
#void libusb_unlock_events(libusb_context *ctx);
libusb_unlock_events = libusb.libusb_unlock_events
libusb_unlock_events.argtypes = [libusb_context_p]
libusb_unlock_events.restype = None
#int libusb_event_handling_ok(libusb_context *ctx);
libusb_event_handling_ok = libusb.libusb_event_handling_ok
libusb_event_handling_ok.argtypes = [libusb_context_p]
#int libusb_event_handler_active(libusb_context *ctx);
libusb_event_handler_active = libusb.libusb_event_handler_active
libusb_event_handler_active.argtypes = [libusb_context_p]
#void libusb_lock_event_waiters(libusb_context *ctx);
libusb_lock_event_waiters = libusb.libusb_lock_event_waiters
libusb_lock_event_waiters.argtypes = [libusb_context_p]
libusb_lock_event_waiters.restype = None
#void libusb_unlock_event_waiters(libusb_context *ctx);
libusb_unlock_event_waiters = libusb.libusb_unlock_event_waiters
libusb_unlock_event_waiters.argtypes = []
libusb_unlock_event_waiters.restype = None
#int libusb_wait_for_event(libusb_context *ctx, struct timeval *tv);
libusb_wait_for_event = libusb.libusb_wait_for_event
libusb_wait_for_event.argtypes = [libusb_context_p, timeval_p]

#int libusb_handle_events_timeout(libusb_context *ctx, struct timeval *tv);
libusb_handle_events_timeout = libusb.libusb_handle_events_timeout
libusb_handle_events_timeout.argtypes = [libusb_context_p, timeval_p]
#int libusb_handle_events_timeout_completed(libusb_context *ctx,
#   struct timeval *tv, int *completed);
try:
    libusb_handle_events_timeout_completed = libusb.\
        libusb_handle_events_timeout_completed
except AttributeError:
    # No safe replacement possible.
    pass
else:
    libusb_handle_events_timeout_completed.argtypes = [
        libusb_context_p, timeval_p, c_int_p]
#int libusb_handle_events(libusb_context *ctx);
libusb_handle_events = libusb.libusb_handle_events
libusb_handle_events.argtypes = [libusb_context_p]
#int libusb_handle_events_completed(libusb_context *ctx, int *completed);
try:
    libusb_handle_events_completed = libusb.libusb_handle_events_completed
except AttributeError:
    # No safe replacement possible.
    pass
else:
    libusb_handle_events_completed.argtypes = [libusb_context_p, c_int_p]
#int libusb_handle_events_locked(libusb_context *ctx, struct timeval *tv);
libusb_handle_events_locked = libusb.libusb_handle_events_locked
libusb_handle_events_locked.argtypes = [libusb_context_p, timeval_p]
#int libusb_get_next_timeout(libusb_context *ctx, struct timeval *tv);
libusb_get_next_timeout = libusb.libusb_get_next_timeout
libusb_get_next_timeout.argtypes = [libusb_context_p, timeval_p]

class libusb_pollfd(Structure):
    _fields_ = [
        ('fd', c_int),
        ('events', c_short),
    ]
libusb_pollfd_p = POINTER(libusb_pollfd)
libusb_pollfd_p_p = POINTER(libusb_pollfd_p)

libusb_pollfd_added_cb_p = CFUNCTYPE(None, c_int, c_short, py_object)
libusb_pollfd_removed_cb_p = CFUNCTYPE(None, c_int, py_object)

#const struct libusb_pollfd **libusb_get_pollfds(libusb_context *ctx);
libusb_get_pollfds = libusb.libusb_get_pollfds
libusb_get_pollfds.argtypes = [libusb_context_p]
libusb_get_pollfds.restype = libusb_pollfd_p_p
#void libusb_set_pollfd_notifiers(libusb_context *ctx,
#        libusb_pollfd_added_cb added_cb, libusb_pollfd_removed_cb removed_cb,
#        void *user_data);
libusb_set_pollfd_notifiers = libusb.libusb_set_pollfd_notifiers
libusb_set_pollfd_notifiers.argtypes = [libusb_context_p,
                                        libusb_pollfd_added_cb_p,
                                        libusb_pollfd_removed_cb_p, py_object]
libusb_set_pollfd_notifiers.restype = None

#typedef int libusb_hotplug_callback_handle;
libusb_hotplug_callback_handle = c_int

libusb_hotplug_flag = Enum({
    'LIBUSB_HOTPLUG_ENUMERATE': 1,
})

libusb_hotplug_event = Enum({
    'LIBUSB_HOTPLUG_EVENT_DEVICE_ARRIVED': 0x01,
    'LIBUSB_HOTPLUG_EVENT_DEVICE_LEFT': 0x02,
})

LIBUSB_HOTPLUG_MATCH_ANY = -1

#typedef int (*libusb_hotplug_callback_fn)(libusb_context *ctx,
#        libusb_device *device, libusb_hotplug_event event, void *user_data);
libusb_hotplug_callback_fn_p = CFUNCTYPE(
    c_int, libusb_context_p, libusb_device_p, c_int, c_void_p)

#int libusb_hotplug_register_callback(libusb_context *ctx,
#        libusb_hotplug_event events, libusb_hotplug_flag flags,
#        int vendor_id, int product_id, int dev_class,
#        libusb_hotplug_callback_fn cb_fn, void *user_data,
#        libusb_hotplug_callback_handle *handle);
try:
    libusb_hotplug_register_callback = libusb.libusb_hotplug_register_callback
except AttributeError:
    pass
else:
    libusb_hotplug_register_callback.argtypes = [
        libusb_context_p,
        c_int, c_int,
        c_int, c_int, c_int,
        libusb_hotplug_callback_fn_p, c_void_p,
        POINTER(libusb_hotplug_callback_handle),
    ]
    libusb_hotplug_register_callback.restype = c_int

#void libusb_hotplug_deregister_callback(libusb_context *ctx,
#        libusb_hotplug_callback_handle handle);
try:
    libusb_hotplug_deregister_callback = \
        libusb.libusb_hotplug_deregister_callback
except AttributeError:
    pass
else:
    libusb_hotplug_deregister_callback.argtypes = [
        libusb_context_p,
        libusb_hotplug_callback_handle,
    ]
    libusb_hotplug_deregister_callback.restype = None

# /libusb.h
