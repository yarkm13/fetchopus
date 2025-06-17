<!-- Google Tag Manager (noscript) -->
<noscript><iframe src="https://www.googletagmanager.com/ns.html?id=GTM-KQKTV9CB"
height="0" width="0" style="display:none;visibility:hidden"></iframe></noscript>
<!-- End Google Tag Manager (noscript) -->
# Fetchopus

A multiprotocol multithread file downloader written in Go, designed to handle large downloads jobs with resume capability.

[![GitHub stars](https://img.shields.io/github/stars/yarkm13/fetchopus?style=social)](https://github.com/yarkm13/fetchopus/stargazers)
[![Latest release](https://img.shields.io/github/v/release/yarkm13/fetchopus?label=latest&color=blue)](https://github.com/yarkm13/fetchopus/releases)
[![Total downloads](https://img.shields.io/github/downloads/yarkm13/fetchopus/total?label=downloads&color=brightgreen)](https://github.com/yarkm13/fetchopus/releases)

<div class="section" id="latest-release">
    <strong>📦 Latest release details:</strong><br />
    <em>Loading...</em>
</div>
### Download binary for your platform:
<table id="downloads-table">
</table>

<script>
    fetch("https://api.github.com/repos/yarkm13/fetchopus/releases/latest")
        .then(res => res.json())
        .then(data => {
            const releaseBody = data.body ? `<div class="release-body">${data.body}</div>` : "<em>No release notes.</em>";
            document.getElementById("latest-release").innerHTML = `
<strong>📦 Latest release details:</strong><br />
<a href="${data.html_url}" target="_blank">${data.name || data.tag_name}</a>
${releaseBody}
`;
            const table = document.getElementById("downloads-table");
            data.assets.forEach(asset => {
                const row = table.insertRow();
                const assetNameCell = row.insertCell(0);
                const curlCommandCell = row.insertCell(1);

                assetNameCell.innerHTML = `<a href="${asset.browser_download_url}">${asset.name}</a>`;
                curlCommandCell.innerHTML = `
<div class="language-plaintext highlighter-rouge"><div class="highlight"><pre class="highlight"><code>
curl -L "${asset.browser_download_url}" -o fetchopus
</code></pre></div></div>
`;
            });
        })
        .catch(err => {
            document.getElementById("latest-release").innerText = "Failed to load latest release notes.";
            console.error(err);
        });
</script>

## Features

- Download files from various type of servers
- Parallel downloads with configurable thread count
- Resume capability through job files
- Recursive directory listing

## Usage

```
./fetchopus --url ftp://user@server.com/path --target-dir /local/path --threads 4
```

To resume a download:

```
./fetchopus --job myjob.dljob
```

## Building

```
make build
```
