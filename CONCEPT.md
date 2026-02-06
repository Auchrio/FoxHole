# Concept for the FoxHole device
**Project Statement**: FoxHole is a stealth VPN deployment for Red Team operations. Leveraging the Luckfox Pico Mini B’s tiny footprint, it serves as a "plug-and-forget" remote access point. It establishes an encrypted outbound WireGuard tunnel to bypass NAT, ensuring persistent, low-observable access to internal LANs.

## Hardware
Since we must factor in lossrate of these devices as an operating cost, the device itself should be both cheap, and easy to construct. Therefore for this project we have decided to use the following hardware:
- Luckfox Pico Mini B USD$13 ([OEM LINK](https://www.luckfox.com/Luckfox-Pico-Mini-A?ci=534))
- RTL8723 WIFI SDIO Expansion Module Board USD$3-7 ([AliExpress](https://www.aliexpress.com/item/1005009569810790.html))

These devices combined lead to a per device cost of around USD$16-20, which is very reasonable, enabling large scale deployments without great fear of device loss.

## Software
On the device itself, it should have the following capabilities:
- Wait for a message from the connecting device in order to initiate a DNS hole punching procedure. (Done through the [pulse tool](https://github.com/Auchrio/pulse))
- Host a wireguard vpn that forwards traffic through its local network, emulating the connecting device really being at the location of the device. (Done through [wireguard-linux-compat](https://github.com/WireGuard/wireguard-linux-compat), or [wireguard-go](https://github.com/WireGuard/wireguard-go))
- Attempt to employ multiple methods of regaining connection if wifi signal is lost such as:
    - Attempting to connect to nearby open wifi signals. (Using a simple go program)
    - Employ an [Evil Twin](https://www.kaspersky.com/resource-center/preemptive-safety/evil-twin-attacks) attack to fish nearby people to provide updated wifi credentials to the device. (using a go program in combination with the [rtl8723bu](https://github.com/lwfinger/rtl8723bu) drivers to allow the RTL8723 WIFI Board to act as an access point)
- Employ a [Keystroke Injection](https://zsecurity.org/glossary/keystroke-injection-usb-or-cable/) attack on unauthorised device that attempts to connect to the device.
- Employ a custom Anti-Analysis script to remove traces and brick firmware when connected to an unauthorised interface. (Go Script that runs on startup)

## Usage
### Usecase One: When the local network is known.
In this case the device will be provided with the network information before being planted and will automatically establish a connection and start listening for outside connections immediately.

In a Red vs Blue team excersise, the device can be planted after the Red team has successfully gained access to the Blue teams local network, allowing them to interact with the services on the local net and use the Blue teams wifi to interact with remote targets that may be whitelisted to only be accessed from the local network's ip.

In the case that the Blue team resets there wifi credentials, whether in an attempt to kick the device from the network or as a side effect of other red team activity. The device will engage [Usecase Two](#usecase-two-when-the-local-network-is-unknown) in order to attempt to reconnect and keep adding value for the Red team.

### Usecase Two: When the local network is unknown
In this case the device will be configured to go staight into fishing mode on boot, where it begins masquerading as the Blue teams local wifi, in this case the device will remain dormant until it has sucessfully fished wifi credentials from a member of the blue team at which point it will remove all trace of the fake access point and begin acting as it would in [Usecase One](#usecase-one-when-the-local-network-is-known)

### Usecase Three: When the device is found by the Blue team.
In this case the device is designed to has one last gift for the Blue team, a [Keystroke Injection](https://zsecurity.org/glossary/keystroke-injection-usb-or-cable/) attack which runs when the device is plugged in by the blue team for analysis, the code that is run by the device can be configured by the user, and can be used to attempt to gain a more stable foothold on the Blue teams internal network.

After the keystroke injection attack is complete, the device runs a cleanup script which does the following:
- Wipes and Overwrites all data on the device to prevent analysis by the blue team
- Bricks the firmware to further waste the blue teams time when they try to analyse it.

## Disclaimer: READ CAREFULLY BEFORE USE
The FoxHole project is designed strictly for authorized Red Team engagements, penetration testing, and cybersecurity research where explicit, written permission has been granted by the network owner. Deploying this device on a network you do not own or have formal authorization to test is illegal and may be subject to criminal prosecution.