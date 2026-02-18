let data = ((% data %));
var infoboxTimer;

function showCommitInfo(target) {
    if (!target || !target.id || !data[target.id]) return;
    const commit = data[target.id];
    document.getElementById("hash").innerHTML = commit.hash;
    const typeEl = document.getElementById("type");
    const scopeEl = document.getElementById("scope");
    if (commit.message.type) { typeEl.style.display = "inline"; typeEl.innerHTML = commit.message.type; } else { typeEl.style.display = "none"; }
    if (commit.message.scope) { scopeEl.style.display = "inline"; scopeEl.innerHTML = commit.message.scope; } else { scopeEl.style.display = "none"; }
    document.getElementById("title").innerHTML = commit.message.title;
    document.getElementById("message").innerHTML = commit.message.body;
    document.getElementById("author").innerHTML = commit.author;
    document.getElementById("committer").innerHTML = commit.committer;
    document.getElementById("authored-date").innerHTML = commit.authored_date_delta;
    document.getElementById("authored-date").setAttribute("title", commit.authored_date);
    document.getElementById("committed-date").innerHTML = commit.committed_date_delta;
    document.getElementById("committed-date").setAttribute("title", commit.committed_date);

    const infobox = document.getElementById("infobox");
    infobox.style.visibility = "visible";
    infobox.style.opacity = "100%";
}

function hideCommitInfo() {
    if (infoboxTimer != null) { clearTimeout(infoboxTimer); infoboxTimer = null; }
    infoboxTimer = setTimeout(() => {
        document.getElementById("infobox").style.opacity = "0%";
        document.getElementById("infobox").style.visibility = "hidden";
        infoboxTimer = null;
    }, 200);
}

window.addEventListener('mouseover', (e) => {
    if (data[e.target.id]) {
        if (infoboxTimer != null) { clearTimeout(infoboxTimer); infoboxTimer = null; }
        const infobox = document.getElementById("infobox");
        const maxY = window.innerHeight - infobox.offsetHeight;
        infobox.style.top = Math.min(e.clientY, maxY) + "px";
        infobox.style.left = e.clientX + 12 + "px";
        showCommitInfo(e.target);
    } else if (e.target.closest && e.target.closest("#infobox")) {
        if (infoboxTimer != null) { clearTimeout(infoboxTimer); infoboxTimer = null; }
    } else {
        hideCommitInfo();
    }
});

window.addEventListener('focusin', (e) => {
    if (data[e.target.id]) {
        if (infoboxTimer != null) { clearTimeout(infoboxTimer); infoboxTimer = null; }
        const infobox = document.getElementById("infobox");
        const rect = e.target.getBoundingClientRect();
        const maxY = window.innerHeight - infobox.offsetHeight;
        infobox.style.top = Math.min(rect.top + rect.height / 2, maxY) + "px";
        infobox.style.left = rect.right + 12 + "px";
        showCommitInfo(e.target);
    }
});

window.addEventListener('focusout', () => { hideCommitInfo(); });
