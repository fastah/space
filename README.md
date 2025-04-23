# Satellite Internet tools

Detecting a Starlink user's geolocation using SpaceX's IP address public data feeds: https://starlink.getfastah.com


## Introduction

The [Starlink by Fastah](https://starlink.getfastah.com) is a web-based tool that helps [Starlink](https://www.starlink.com) (and Viasat) satellite internet users detect whether their IP address is being correctly advertised for geolocation by Starlink's public IP Geolocation data feeds. This tool is useful for Starlink users who are experiencing issues with geolocation-based services, such as streaming services, online banking, and other services that rely on accurate geolocation data.

This tool is made by Fastah, a startup that builds software API tools to help developers use IP-based geolocation and security quickly and accurately. 

This project is *not* supported or endorsed by SpaceX, Starlink, or any satellite internet service in any way. All data is sourced from publicly available data feeds provided by SpaceX and Viasat using the [RFC8805 internet standard](https://www.rfc-editor.org/rfc/rfc8805).

## How it works

### This is not GPS-based geolocation!

We use Fastah's IP-based Geolocation API ([AWS](https://aws.amazon.com/marketplace/pp/B084VR96P3), [Azure](https://azuremarketplace.microsoft.com/en-us/marketplace/apps/fastah.ip_location_api_01)) to detect the geolocation of your IP address using the raw underlying data provided by SpaceX and Viasat themselves. Both SpaceX and Viasat provide [RFC8805 internet standard](https://www.rfc-editor.org/rfc/rfc8805) feeds in the public domain so that app developers and network operators build services such as Netflix, Prime Video, online shopping, and banking services more efficiently and securely. 

In an upcoming release, we will allow the app's users to flag wrong data based on real (GPS) location of the user. This will help SpaceX and Viasat improve their geolocation data feeds and make the internet a better place for everyone.

## How to use

1. Visit the app: <https://starlink.getfastah.com>
2. Let the web app scan your public ISP-assigned IP address; Fastah's API web service to lookup IP-bsaed geolocation without storing any personally-identifiable information.
3. Enjoy looking at your Starlink-advertised geolocation on the world map. 
4. if you aren't browsing using Starlink connection, you are allowed to simulate an Starlink IP address by clicking "Simulate Starlink IP" button, and see your simulated geolocation on the map.

## Attribution

1. Wikipedia: for SVG brand logos for [Starlink](https://en.m.wikipedia.org/wiki/File:Starlink_Logo.svg) and [Viasat](https://en.m.wikipedia.org/wiki/File:Starlink_Logo.svg). 
2. SpaceX and Starlink are trademarks of SpaceX, Inc. Viasat is a trademark of Viasat, Inc.

## License
Most *data* files in [this Github repository](https://github.com/fastah/space/) are built from public domain sources, so they are licensed under Creative Commons Attribution Share Alike 4.0 International ([CC BY-SA 4.0](https://choosealicense.com/licenses/cc-by-sa-4.0/)). 

The *code* is licensed under the [MIT License](https://choosealicense.com/licenses/mit/).

## All rights reserved

Fastah™ is a trademark of Blackbuck Computing, Inc., a Delaware corporation. Contact us at support@getfastah.com. 
