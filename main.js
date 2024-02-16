const ctaButton = document.getElementById("scanIpButton");
const ctaExplainer = document.getElementById("ctaExplainer");
const regionNamesEnglish = new Intl.DisplayNames(["en"], { type: "region" });

let allFeedMeta = {};
let allFeedSamples = {};

// Loads the JSON metadata file for the RFC8805 feed from the specified provider 
async function initRfc8805meta(provider) {
    try {
        const requestURL = "./gen/latest-feeds/" + provider + "/rfc8805.meta.json";
        const request = new Request(requestURL);
        const response = await fetch(request);
        const feedmeta = await response.json();
        const lmt = Date.parse(feedmeta["lastModified"]);
        console.log("[" + provider + "] RFC8805 metadata lastModified:" + new Date(lmt));
        allFeedMeta[provider] = feedmeta;
    } catch (error) {
        console.error("[", provider, "] RFC8805 metadata read error:", error);
    }
}

// Loads sample IP addresses for this specified provider 
async function initRfc8805sampleIp(provider) {
    try {
        const requestURL = "./gen/latest-feeds/" + provider + "/samples.json";
        const request = new Request(requestURL);
        const response = await fetch(request);
        const samples = await response.json();
        allFeedSamples[provider] = samples;
    } catch (error) {
        console.error("[", provider, "] IP samples read error:", error);
    }
}

// Loads the IP-metadata using Fastah API's endpoint
async function getIPGeolocation(searchIp = 'auto') {
    fetch('https://fastah-v2.azure-api.net/whereis/v1/json/' + searchIp, {
        mode: 'cors',
        headers: {
            'Fastah-Key': '39d645630da04953b52a29bbb7ad3903'
        }
    })
    .then((response) => {
        if (!response.ok) {
            console.log("Error response from Fastah API (likely CORS?): " + response.status + " " + response.statusText);
            return null;
        }
        return response.json();
    })
    .then(
        data => {
            const lce = new CustomEvent("locationchanged", {
                detail: {
                  locationSrc: "real-ipapi",
                  ip: data.ip,
                  ld: data.locationData,
                  sd: Object.hasOwn(data, 'satellite') ? data.satellite : null,
                },
              });
            document.dispatchEvent(lce);
        }
    )
    .catch(error => console.error('Error:', error));
}

// Disables the button and changes the supplementary text to indicate additional user context
function disableButton(newButtonText, explainerText) {
    ctaButton.disabled = true;
    ctaButton.innerText = newButtonText;
    ctaButton.classList.remove("btn-primary");
    ctaButton.classList.add("btn-secondary");
    ctaButton.removeAttribute('data-opmode');
    ctaExplainer.innerText = explainerText;
}

// (Re)enables the button and changes the supplementary text to indicate additional user context
function enableButton(newButtonText, explainerText, opmode = "feeling-lucky") {
    ctaButton.innerText = newButtonText;
    ctaButton.classList.remove("btn-secondary");
    ctaButton.classList.add("btn-primary");
    ctaExplainer.innerText = explainerText;
    ctaButton.disabled = false;
    ctaButton.setAttribute('data-opmode', opmode);
}

function simulateRandomLocation(provider = 'starlink', fallbackIp = '98.97.5.1', preferCountry = 'US') {
    simulatedCountry = preferCountry
    allFeedMeta[provider].visibleCountries.forEach(cc => {
        if (preferCountry == cc) { 
            console.log("Found preferred country in this providers feed country " + preferCountry);
            simulatedCountry = cc;
        }
    }  );
    let samples = null
    for (const feat of allFeedSamples[provider].features) {
        if (feat.properties["cciso2"] === simulatedCountry) {
            samples = feat.properties["ip-samples"]
        }
    }
    if (samples === null) {
        return 
    }
    console.log(samples);
    fallbackIp = samples[Math.floor(Math.random() * samples.length)];
    const randomLocationChange = new CustomEvent("locationchanged", {
        detail: {
            locationSrc: "simulated",
            ip: fallbackIp,
            cciso2: simulatedCountry,
            ld: { countryCode: simulatedCountry, countryName: regionNamesEnglish.of(simulatedCountry) },
            sd: { provider : provider },
        },
      });
    console.log("Sending simulated location change event: %O", randomLocationChange);
    document.dispatchEvent(randomLocationChange);
}

function updateUIwithNewLocation(event) {
    if (event.type != "locationchanged") {
        console.log("Not a location change event?");
        return;
    }
    if (!('sd' in event.detail) || event.detail.sd == null) {
        console.log("Not a starlink IP address");
        let explainerText = "You are not on Starlink";
        if ('ld' in event.detail) {
            explainerText += ", with an IP address " + event.detail.ip + " in " + event.detail.ld.countryName + ".";    
        }
        enableButton("Simulate a random Starlink location", explainerText, "feeling-lucky")
    } else {
        console.log("Starlink IP address");
        explainerText = "Your Starlink IP is " + event.detail.ip + " in " + event.detail.ld.countryName + ".";    
        enableButton("Show me a random Starlink location", explainerText, "feeling-lucky")
    }
}

ctaButton.addEventListener("click", (event) => {
    console.log(event);
    if (ctaButton.getAttribute('data-opmode') === "feeling-lucky") {
        simulateRandomLocation('starlink');
    } else { 
        console.log("No data-attribute configured on button?");
    }
  });


function initMaps() { 
    // This is a browser-side API key, and it's to have it in public Git repos. It should be ideally configured to only serve from the domain of this website.
    mapboxgl.accessToken = 'pk.eyJ1IjoiczhtYXRodXIiLCJhIjoiY2xzbHl1Zjg0MGZpdjJrcGVpa2pkbG0wNiJ9.rFDdt45Wd4s6a-RfqvAQiQ';
    const map = new mapboxgl.Map({
        container: 'map',
        // Choose from Mapbox's core styles, or make your own style with Mapbox Studio
        style: 'mapbox://styles/mapbox/streets-v12',
        center: [-96, 37.8],
        zoom: 3,
        maxZoom: 4,
        minZoom: 2
    });
    
    map.on('load', () => {
        // Add an image to use as a custom marker
        map.loadImage(
            './static/marker-icons/mapbox-marker-icon-20px-gray.png',
            (error, image) => {
                if (error) throw error;
                map.addImage('custom-marker', image);
    
                map.addSource('starlink', {
                    type: 'geojson',
                    // Use a URL for the value for the `data` property.
                    data: './gen/latest-feeds/starlink/samples.json'
                });

                map.addSource('viasat', {
                    type: 'geojson',
                    // Use a URL for the value for the `data` property.
                    data: './gen/latest-feeds/viasat/samples.json'
                });

                // Add a symbol layer
                map.addLayer({
                    'id': 'starlink-layer',
                    'type': 'symbol',
                    'source': 'starlink',
                    'layout': {
                        'icon-image': 'custom-marker', // reference the image
                        'icon-size': 1.0
                        }
                });
            }
        );
    });    
}    
// This is automatically done via Fastah API on page load
initRfc8805meta("starlink");
initRfc8805sampleIp("starlink");
disableButton("Scanning...", "We are checking if you are using a Starlink IP", "processing");
document.addEventListener("locationchanged", (event) => {
    console.log("Updating UI with new location: %O", event);
    updateUIwithNewLocation(event);
});
getIPGeolocation();
initMaps();
