# FoxHole
FoxHole is a stealth VPN deployment for Red Team operations. Leveraging the Luckfox Pico Mini B’s tiny footprint, it serves as a "plug-and-forget" remote access point. It establishes an encrypted outbound WireGuard tunnel to bypass NAT, ensuring persistent, low-observable access to internal LANs.

## Development
Currently the only portion of the FoxHole project that is developed is the [pulse tool](https://github.com/Auchrio/pulse) which is going to be used to provide a serverless solution to communicating with the harware in situations where any and all devices are subjected to CGNAT or Hard NAT systems.

If you are interested in what this device will be capable when complete, please read the [CONCEPT.md document](CONCEPT.md)

## Disclaimer: READ CAREFULLY BEFORE USE
The FoxHole project is designed strictly for authorized Red Team engagements, penetration testing, and cybersecurity research where explicit, written permission has been granted by the network owner. Deploying this device on a network you do not own or have formal authorization to test is illegal and may be subject to criminal prosecution.