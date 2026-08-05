const token = new URLSearchParams(window.location.search).get("token");

const loading = document.getElementById("loading");
const content = document.getElementById("content");
const error = document.getElementById("error");

const passwordSection = document.getElementById("passwordSection");
const passwordInput = document.getElementById("password");
const downloadForm = document.getElementById("downloadForm");
const downloadBtn = document.getElementById("downloadBtn");

let requiresPassword = false;

function formatFileSize(bytes) {
    if (bytes === 0) return "0 B";

    const units = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));

    return `${(bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

function showError(message) {
    error.textContent = message;
    error.style.display = "block";
}

function hideError() {
    error.style.display = "none";
}

if (!token) {
    loading.style.display = "none";
    showError("No share token provided.");
} else {
    fetch(`/api/share/${token}`)
        .then(async res => {
            if (!res.ok) {
                throw await res.json().catch(() => ({ error: "Unable to load share." }));
            }
            return res.json();
        })
        .then(data => {
            loading.style.display = "none";
            content.style.display = "block";

            document.getElementById("filename").textContent = data.filename;
            document.getElementById("filesize").textContent = formatFileSize(data.size);

            requiresPassword = data.password_required;

            if (requiresPassword) {
                passwordSection.style.display = "block";
                passwordInput.required = true;
            }
        })
        .catch(err => {
            loading.style.display = "none";
            showError(err.error || "Unable to load share.");
        });
}

downloadForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    hideError();

    downloadBtn.disabled = true;
    downloadBtn.textContent = "Downloading...";

    const body = {};
    if (requiresPassword) {
        body.password = passwordInput.value;
    }

    try {
        const res = await fetch(`/api/share/${token}/download`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body)
        });

        if (!res.ok) {
            const err = await res.json().catch(() => ({}));
            showError(err.error || "Download failed.");
            return;
        }

        const blob = await res.blob();

        const disposition = res.headers.get("Content-Disposition") || "";
        const match = disposition.match(/filename\*?=(?:UTF-8'')?"?([^";]+)"?/i);
        const filename = match ? decodeURIComponent(match[1]) : document.getElementById("filename").textContent;

        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        a.remove();
        URL.revokeObjectURL(url);

    } catch (err) {
        showError("Network error. Please try again.");
    } finally {
        downloadBtn.disabled = false;
        downloadBtn.textContent = "Download";
    }
});