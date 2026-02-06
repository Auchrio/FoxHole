# Concept for the FoxHole device
**Project Statement**: FoxHole is a stealth VPN deployment for Red Team operations. Leveraging the Luckfox Pico Mini B’s tiny footprint, it serves as a "plug-and-forget" remote access point. It establishes an encrypted outbound WireGuard tunnel to bypass NAT, ensuring persistent, low-observable access to internal LANs.

## Hardware
Since we must factor in loss rate of these devices as an operating cost, the device itself should be both cheap and easy to construct. Therefore, for this project we have decided to use the following hardware:
- Luckfox Pico Mini B USD$13 ([OEM LINK](https://www.luckfox.com/Luckfox-Pico-Mini-A?ci=534))
- RTL8723 WIFI SDIO Expansion Module Board USD$3-7 ([AliExpress](https://www.aliexpress.com/item/1005009569810790.html))

These devices combined lead to a per-device cost of around USD$16-20, which is very reasonable, enabling large-scale deployments without great fear of device loss.

## Software
On the device itself, it should have the following capabilities:
- Listen for a message from the connecting device to initiate DNS hole punching via the [pulse tool](https://github.com/Auchrio/pulse)
- Host a WireGuard VPN that forwards traffic through its local network, making the connecting device appear to be at the device's location via [wireguard-linux-compat](https://github.com/WireGuard/wireguard-linux-compat) or [wireguard-go](https://github.com/WireGuard/wireguard-go)
- Employ multiple connection recovery methods if WiFi signal is lost:
    - Attempt to connect to nearby open WiFi networks (using a simple Go program)
    - Deploy an [Evil Twin](https://www.kaspersky.com/resource-center/preemptive-safety/evil-twin-attacks) attack to trick nearby users into providing WiFi credentials (using a Go program with [rtl8723bu](https://github.com/lwfinger/rtl8723bu) drivers to enable the RTL8723 WiFi board to act as an access point)
- Execute a [Keystroke Injection](https://zsecurity.org/glossary/keystroke-injection-usb-or-cable/) attack on any unauthorized device attempting to connect
- Run a custom anti-analysis script on startup to remove traces and brick firmware if connected to an unauthorized interface

## Usage
### Use Case One: When the local network is known
In this case the device will be provided with the network information before being planted and will automatically establish a connection and start listening for outside connections immediately.

In a Red vs Blue team exercise, the device can be planted after the Red team has successfully gained access to the Blue team's local network, allowing them to interact with services on the local network and use the Blue team's WiFi to interact with remote targets that may be whitelisted to only be accessed from the local network's IP.

If the Blue team resets their WiFi credentials—whether to disconnect the device or as a side effect of other Red team activity—the device will engage [Use Case Two](#use-case-two-when-the-local-network-is-unknown) to attempt reconnection and continue providing value to the Red team.

### Use Case Two: When the local network is unknown
In this case, the device will be configured to enter fishing mode on startup, masquerading as the Blue team's local WiFi. It will remain dormant until it successfully obtains WiFi credentials from a Blue team member. Once obtained, it will remove all traces of the fake access point and begin operating as in [Use Case One](#use-case-one-when-the-local-network-is-known).

### Use Case Three: When the device is found by the Blue team
In this final scenario, the device has one last defense: a [Keystroke Injection](https://zsecurity.org/glossary/keystroke-injection-usb-or-cable/) attack that executes when the device is plugged in by the Blue team for analysis. The injected code can be configured by the user and may be used to establish a more stable foothold on the Blue team's internal network.

After the keystroke injection attack completes, the device runs a cleanup script that:
- Wipes and overwrites all data on the device to prevent analysis by the Blue team
- Bricks the firmware to hinder any further analysis attempts by the Blue team

## Disclaimer: READ CAREFULLY BEFORE USE
The FoxHole project is designed strictly for authorized Red Team engagements, penetration testing, and cybersecurity research where explicit, written permission has been granted by the network owner. Deploying this device on a network you do not own or have formal authorization to test is illegal and may be subject to criminal prosecution.