"strict mode";

var r = new XMLHttpRequest();
r.open("GET", "/count", true);
r.onreadystatechange = function () {
    if (r.readyState !== 4) return;
    
    var documentsCount = document.getElementById("documents-count");
    if (r.status !== 200) {
        doumentsCount = "Unable to connect to the server";
        return;
    }
    var data = JSON.parse(r.responseText);
    documentsCount.firstChild.data = 
        data.count + " magnets indexed.";
};
r.send();

var nbSearches = 0;
function searchDocuments(q) {
    document.body.classList.add("with-results");
    var infos = document.getElementById("results-infos")
    infos.style.display = 'block';
    infos.firstChild.data = "Loading…";
    var currentNbSearches = ++nbSearches;
    var query = new XMLHttpRequest();
    query.open("GET", "/search?q="+encodeURIComponent(q), true);
    query.onreadystatechange = function() {
        if (query.readyState !== 4) return;

        if (query.status !== 200) {
            alert("Unable to load documents.")
            return;
        }
        if (currentNbSearches !== nbSearches) {
            //console.log("wat")
            // too fast
            return;
        }
        var data = JSON.parse(query.responseText);
        displayDocuments(data);
    };
    query.send();
}

function bytesToSize(bytes) {
    var sizes = ['bytes', 'Kb', 'Mb', 'Gb', 'Tb'];
    if (bytes === 0) return '0 bytes';
    var i = parseInt(Math.floor(Math.log(bytes) / Math.log(1024)));
    return Math.round(bytes / Math.pow(1024, i), 2) + ' ' + sizes[i];
}

function expandFiles(files) {
    //console.log(this, files)

    var filesList = this.lastChild;
    if (filesList.className !== "files-list") {
        filesList = document.createElement("table");
        filesList.className = "files-list";
        for (var i = 0, l = files.length; i < l; ++i) {
            var fileRow = document.createElement("tr");
            var nameTd = document.createElement("td");
            nameTd.appendChild(document.createTextNode(files[i].path));
            var sizeTd = document.createElement("td");
            sizeTd.className = "file-size";
            sizeTd.appendChild(document.createTextNode(bytesToSize(files[i].size)));
            fileRow.appendChild(nameTd);
            fileRow.appendChild(sizeTd);
            filesList.appendChild(fileRow);
        }
        this.appendChild(filesList);
    } else {
        this.removeChild(filesList);
    }
}

function displayDocuments(data) {
    var documents = data.results;
    var count = data.totalHits;

    var list = document.getElementById("results");
    var infos = document.getElementById("results-infos")

    if (count > 0) {
        for (var i = 0, l = documents.length; i < l; ++i) {
            var doc = documents[i];
            var li = document.createElement("li");
            var buttonsDiv = document.createElement("div");
            buttonsDiv.className = "document-details";

            var linkHref = "magnet:?xt=urn:btih:"+doc.hash+"&dn="+encodeURIComponent(doc.name);
            var magnetLink = document.createElement("a");
            magnetLink.setAttribute("href", linkHref);
            magnetLink.className = "magnet-link";
            //magnetLink.appendChild(document.createTextNode("\u2B8B Download"))
            magnetLink.appendChild(document.createTextNode("🔗"))

            var mainLink = document.createElement("a");
            mainLink.setAttribute("href", linkHref);
            mainLink.className = "main-link";
            mainLink.appendChild(document.createTextNode(doc.name));
            
            var sizeInfo = document.createElement("span");
            sizeInfo.className = "size-info";
            sizeInfo.appendChild(document.createTextNode(bytesToSize(doc.size)));
            buttonsDiv.appendChild(sizeInfo);

            var searchLink = document.createElement("a");
            searchLink.setAttribute("href", "https://qwant.com/?t=web&q=\""+doc.hash+"\"");
            searchLink.className = "search-link";
            //searchLink.appendChild(document.createTextNode("\u24E0 Search"))
            searchLink.appendChild(document.createTextNode("🔎"))
            
            var filesButton = document.createElement("button");
            //filesButton.appendChild(document.createTextNode("files"))
            filesButton.appendChild(document.createTextNode("🗄"))
            filesButton.addEventListener("click", expandFiles.bind(li, doc.files));

            buttonsDiv.appendChild(searchLink);
            buttonsDiv.appendChild(filesButton);

            

            li.appendChild(magnetLink);
            li.appendChild(mainLink);
            li.appendChild(buttonsDiv);

            list.appendChild(li);
        }
        if (count !== documents.length) {
            infos.firstChild.data = documents.length + " magnets, from the "+count+" magnets found.";
        } else {
            infos.firstChild.data = count + " magnet"+(count > 1 ? "s" : "")+" found.";
        }
    } else {
        infos.firstChild.data = "No magnets found.";
    }

}

var getParams = new URLSearchParams(window.location.search);
if (getParams.has('q')) {
    var q = getParams.get('q');
    document.getElementById("q").value = q;
    searchDocuments(q)
}

var lastSearch = 0;
document.getElementById("search-form")
.addEventListener("submit", function(e) {
    e.preventDefault();

    // Search value
    var q = this.q.value;
    
    // Empty results
    var list = document.getElementById("results");
    var infos = document.getElementById("results-infos")
    list.innerHTML = "";
    //infos.firstChild.data = "";

    console.log("wat")
    

    if (!q) {
        document.body.classList.remove("with-results");
        history.pushState(undefined, "Magnets Search Engine", "/");
        return;   
    }
    history.pushState(undefined, q + " | Magnets Search Engine", "/?q="+encodeURIComponent(q));
    var now = +new Date();
    if (now-lastSearch < 500) {
        //console.log("slow down");
        infos.firstChild.data = "slow down";
        return;
    }
    lastSearch = now;
    searchDocuments(q);
});