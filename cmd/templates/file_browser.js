const searchInput = document.getElementById("fileSearch");
const items = Array.from(document.querySelectorAll(".file-item"));
const searchEmpty = document.getElementById("searchEmpty");
const contextMenu = document.getElementById("contextMenu");
const menuOpen = document.getElementById("menu-open");
const menuDownload = document.getElementById("menu-download");
const menuOpenFolder = document.getElementById("menu-open-folder");
const menuOpenLocation = document.getElementById("menu-open-location");
const menuCopy = document.getElementById("menu-copy");

function hideContextMenu() {
    contextMenu.classList.remove("active");
    contextMenu.setAttribute("aria-hidden", "true");
}

function showContextMenu(event, item) {
    event.preventDefault();
    event.stopPropagation();
    const isDir = item.dataset.isDir === "true" || item.dataset.parent === "true";
    const url = item.dataset.url || "";
    const downloadUrl = item.dataset.downloadUrl || "";
    const folderUrl = item.dataset.folderUrl || url;
    const systemPath = item.dataset.systemPath || "";
    const systemFolder = item.dataset.systemFolder || systemPath;

    menuOpen.style.display = url ? "block" : "none";
    menuOpen.textContent = isDir ? "Open" : "Open in browser";
    menuOpen.onclick = () => {
        if (url) {
            window.location.href = url;
        }
        hideContextMenu();
    };

    menuDownload.style.display = downloadUrl ? "block" : "none";
    menuDownload.onclick = () => {
        if (downloadUrl) {
            window.location.href = downloadUrl;
        }
        hideContextMenu();
    };

    menuOpenFolder.style.display = !isDir && folderUrl ? "block" : "none";
    menuOpenFolder.onclick = () => {
        if (folderUrl) {
            window.location.href = folderUrl;
        }
        hideContextMenu();
    };

    menuOpenLocation.style.display = systemFolder ? "block" : "none";
    menuOpenLocation.onclick = () => {
        if (systemFolder) {
            openInExplorer(systemFolder);
        }
        hideContextMenu();
    };

    menuCopy.style.display = systemPath ? "block" : "none";
    menuCopy.onclick = () => {
        if (systemPath) {
            copyToClipboard(decodeURIComponent(systemPath), menuCopy);
        }
        hideContextMenu();
    };

    contextMenu.style.left = "-9999px";
    contextMenu.style.top = "-9999px";
    contextMenu.classList.add("active");
    contextMenu.setAttribute("aria-hidden", "false");
    const rect = contextMenu.getBoundingClientRect();
    const left = Math.min(event.clientX, window.innerWidth - rect.width - 8);
    const top = Math.min(event.clientY, window.innerHeight - rect.height - 8);
    contextMenu.style.left = `${Math.max(8, left)}px`;
    contextMenu.style.top = `${Math.max(8, top)}px`;
}

if (searchInput) {
    searchInput.addEventListener("input", (event) => {
        const query = event.target.value.toLowerCase();
        let visibleCount = 0;
        items.forEach((item) => {
            const name = item.dataset.name.toLowerCase();
            const visible = name.includes(query);
            item.style.display = visible ? "" : "none";
            if (visible) {
                visibleCount += 1;
            }
        });
        searchEmpty.style.display = visibleCount === 0 ? "block" : "none";
    });
}

items.forEach((item) => {
    item.addEventListener("contextmenu", (event) => showContextMenu(event, item));
    const trigger = item.querySelector(".action-trigger");
    if (trigger) {
        trigger.addEventListener("click", (event) => showContextMenu(event, item));
    }
});

document.addEventListener("click", (event) => {
    if (!contextMenu.contains(event.target)) {
        hideContextMenu();
    }
});

document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
        hideContextMenu();
    }
});

setupCopyButtons();
