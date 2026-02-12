const defaultValues = Object.assign(
    { common: {}, root: {}, intermediate: {}, certificate: {} },
    window.dashboardDefaults || {}
);

document.querySelectorAll("[data-fill]").forEach((button) => {
    button.addEventListener("click", () => {
        const form = button.closest("form");
        const target = button.dataset.fill;
        const values = { ...defaultValues.common, ...(defaultValues[target] || {}) };
        Object.entries(values).forEach(([name, value]) => {
            const field = form.querySelector('[name="' + name + '"]');
            if (field) {
                field.value = value;
            }
        });
    });
});

const navItems = document.querySelectorAll(".nav-item");
const sections = document.querySelectorAll(".panel-section");
const validSections = new Set(
    Array.from(navItems)
        .map((item) => item.dataset.section)
        .filter(Boolean)
);

function activateSection(target) {
    navItems.forEach((nav) => nav.classList.toggle("active", nav.dataset.section === target));
    sections.forEach((section) => {
        section.classList.toggle("active", section.dataset.section === target);
    });
}

navItems.forEach((item) => {
    item.addEventListener("click", () => {
        const target = item.dataset.section;
        if (!target) {
            return;
        }
        activateSection(target);
        history.replaceState(null, "", `#${target}`);
    });
});

const initialSection = window.location.hash.slice(1);
if (initialSection && validSections.has(initialSection)) {
    activateSection(initialSection);
}

const certContextMenu = document.getElementById("certContextMenu");
const certMenuDownload = document.getElementById("cert-menu-download");
const certMenuOpenFolder = document.getElementById("cert-menu-open-folder");
const certMenuOpenLocation = document.getElementById("cert-menu-open-location");
const certMenuCopy = document.getElementById("cert-menu-copy");

function hideCertMenu() {
    if (!certContextMenu) {
        return;
    }
    certContextMenu.classList.remove("active");
    certContextMenu.setAttribute("aria-hidden", "true");
}

function showCertMenu(event, row) {
    if (!certContextMenu || !row) {
        return;
    }
    event.preventDefault();
    event.stopPropagation();
    const downloadUrl = row.dataset.downloadUrl || "";
    const folderUrl = row.dataset.folderUrl || "";
    const systemPath = row.dataset.systemPath || "";
    const systemFolder = row.dataset.systemFolder || "";

    certMenuDownload.style.display = downloadUrl ? "block" : "none";
    certMenuDownload.onclick = () => {
        if (downloadUrl) {
            window.location.href = downloadUrl;
        }
        hideCertMenu();
    };

    certMenuOpenFolder.style.display = folderUrl ? "block" : "none";
    certMenuOpenFolder.onclick = () => {
        if (folderUrl) {
            window.location.href = folderUrl;
        }
        hideCertMenu();
    };

    certMenuOpenLocation.style.display = systemFolder ? "block" : "none";
    certMenuOpenLocation.onclick = () => {
        if (systemFolder) {
            openInExplorer(systemFolder);
        }
        hideCertMenu();
    };

    certMenuCopy.style.display = systemPath ? "block" : "none";
    certMenuCopy.onclick = () => {
        if (systemPath) {
            copyToClipboard(decodeQueryValue(systemPath), certMenuCopy);
        }
        hideCertMenu();
    };

    certContextMenu.style.left = "-9999px";
    certContextMenu.style.top = "-9999px";
    certContextMenu.classList.add("active");
    certContextMenu.setAttribute("aria-hidden", "false");
    const rect = certContextMenu.getBoundingClientRect();
    const left = Math.min(event.clientX, window.innerWidth - rect.width - 8);
    const top = Math.min(event.clientY, window.innerHeight - rect.height - 8);
    certContextMenu.style.left = `${Math.max(8, left)}px`;
    certContextMenu.style.top = `${Math.max(8, top)}px`;
}

document.querySelectorAll(".certificate-row").forEach((row) => {
    const trigger = row.querySelector(".action-trigger");
    if (trigger) {
        trigger.addEventListener("click", (event) => showCertMenu(event, row));
    }
});

document.addEventListener("click", (event) => {
    if (certContextMenu && !certContextMenu.contains(event.target)) {
        hideCertMenu();
    }
});

document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
        hideCertMenu();
    }
});

setupCopyButtons();
setupOpenButtons();
