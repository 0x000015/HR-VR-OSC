# HR-VR-OSC


## Heartrate to VRChat's OSC

* Spotify supported
  - Requires a Windows OS and the Spotify Desktop application to be used

## Notes
* Default config will use port 9000 for VRChat OSC's receiving port. 
  - You can change which port VRChat will listen on by setting your launch options to include `--osc=9000:127.0.0.1:9001` and then changing 9000 to any port that is unused.
* HeartRateSource in the config only supports two options. `PULSOID` and your choice of URL as long as it includes `http/http` in the front. EX: `https://example.com/hr`
  - Any URL can be used as long as the URL outputs a number or a string **ONLY**.
  
## Credits
[vrcosc-magicchatbox](https://github.com/BoiHanny/vrcosc-magicchatbox) for their Trend function