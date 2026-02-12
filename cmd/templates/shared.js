const FEEDBACK_TIMEOUT_MS = 1200;

function ensureClipboardLiveRegion() {
    let region = document.getElementById("clipboard-status");
    if (!region) {
        region = document.createElement("div");
        region.id = "clipboard-status";
        region.className = "visually-hidden";
        region.setAttribute("aria-live", "polite");
        region.setAttribute("aria-atomic", "true");
        document.body.appendChild(region);
    }
    return region;
}

function announceStatus(message) {
    const region = ensureClipboardLiveRegion();
    region.textContent = message;
    if (message) {
        setTimeout(() => {
            region.textContent = "";
        }, FEEDBACK_TIMEOUT_MS);
    }
}

function decodeQueryValue(raw) {
    if (!raw) {
        return "";
    }
    return decodeURIComponent(raw.replace(/\+/g, "%20"));
}

function copyToClipboard(text, button) {
    if (!text) {
        return;
    }
    navigator.clipboard.writeText(text).then(() => {
        if (button) {
            const original = button.textContent;
            const originalLabel = button.getAttribute("aria-label");
            button.textContent = "Copied";
            if (originalLabel) {
                button.setAttribute("aria-label", "Copied");
            }
            setTimeout(() => {
                button.textContent = original;
                if (originalLabel) {
                    button.setAttribute("aria-label", originalLabel);
                }
            }, FEEDBACK_TIMEOUT_MS);
        }
        announceStatus("Copied to clipboard");
    }).catch((error) => {
        console.error("Failed to copy to clipboard", error);
        announceStatus("Failed to copy to clipboard");
    });
}

function setupCopyButtons() {
    document.querySelectorAll("[data-copy]").forEach((button) => {
        button.addEventListener("click", async () => {
            const raw = button.getAttribute("data-copy");
            if (!raw) {
                return;
            }
            const text = decodeQueryValue(raw);
            copyToClipboard(text, button);
        });
    });
}

function openInExplorer(rawPath) {
    if (!rawPath) {
        return;
    }
    const path = decodeQueryValue(rawPath);
    fetch("/open", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        body: new URLSearchParams({ path })
    }).then((response) => {
        if (response.ok) {
            announceStatus("Opened folder in file manager");
        } else {
            announceStatus("Failed to open folder in file manager. Check if the folder exists.");
        }
    }).catch(() => {
        announceStatus("Failed to open folder in file manager. Check if the folder exists.");
    });
}

function setupOpenButtons() {
    document.querySelectorAll("[data-open]").forEach((button) => {
        button.addEventListener("click", () => {
            openInExplorer(button.getAttribute("data-open"));
        });
    });
}
