Android Support
---------------

To setup a bot to run one Swarming that 'owns' all the Android devices
connected:

First, copy the udev rule file:

    sudo cp 51-android.rules /etc/udev/rules.d/

Second, make sure the current user that runs the swarming bot is member of
plugdev. If not, run:

    sudo gpasswd -a username plugdev


http://wiki.cyanogenmod.org/w/UDEV has a nice list of most android devices, the
sample file only contain the PCI vendor ID for Google/Nexus devices.
