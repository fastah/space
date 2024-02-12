

async function fetchRfc8805LastModifiedDate(provider) {
    try {
        const requestURL = "latest-feed/" + provider + "/rfc8805.meta.json";
        const request = new Request(requestURL);
        const response = await fetch(request);
        const feedmeta = await response.json();
        const lmt = Date.parse(feedmeta["last-modified-by-publisher"]);
    } catch (error) {
        console.error("Starlink RFC8805 metadata read error:", error);
    }
  }

  fetchRfc8805LastModifiedDate("starlink")
  