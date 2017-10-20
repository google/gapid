function openTab(link, content) {
    var i, tabcontent, tablinks;

    tabcontent = document.getElementsByClassName("tabcontent");
    for (i = 0; i < tabcontent.length; i++) {
        tabcontent[i].className = (content == tabcontent[i]) ? "tabcontent active" : "tabcontent";
    }

    tablinks = document.getElementsByClassName("tablink");
    for (i = 0; i < tablinks.length; i++) {
        tablinks[i].className = (link == tablinks[i]) ? "tablink active" : "tablink";
    }
}

function bindTabLinkClick(link, content) {
    link.addEventListener("click", function(ev) {
        openTab(link, content);
    })
}

window.onload = function() {
    var tabs, tabbar, first;
    first = true;

    tabbar = document.createElement("div");
    tabbar.className = "tabbar";

    tabs = document.getElementsByClassName("tab");
    while (tabs.length > 0) { // The collection is live, and we're changing class names.
        var tab, link;

        tab = tabs[0];

        link = document.createElement("button");
        link.className = "tablink";
        link.innerText = tab.id;
        tabbar.appendChild(link);

        if (first) {
            // Insert the tab-bar before the first tab.
            tab.parentElement.insertBefore(tabbar, tab);
            tab.className = "tabcontent active";
            first = false;
        } else {
            tab.className = "tabcontent";
        }

        bindTabLinkClick(link, tab);
    }
}
