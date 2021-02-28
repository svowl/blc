var reports = [];

document.addEventListener('authPassed', (event) => {
    loadReports();
});

function loadReports() {
    lockBlock(document.getElementById("reports"));
    let xhr = new XMLHttpRequest();
    xhr.open('POST', '/api/reports/' + authToken);
    xhr.setRequestHeader('Content-type', 'application/json; charset=utf-8');
    xhr.send();
    xhr.onload = function () {
        if (xhr.status != 200) {
            console.log('XHR error: ' + xhr.status);
            return;
        }
        console.log(xhr.response);
        if (xhr.response) {
            let data = JSON.parse(xhr.response);
            let pattern = document.getElementById("report-pattern");
            reports = [];
            if (pattern) {
                // Toggle no-reports block visibility
                if (data.length > 0) {
                    document.getElementById("no-reports").style.display = 'none';
                } else {
                    document.getElementById("no-reports").style.display = 'block';
                }
                // Remove all current reports
                let reportsBlockChildren = document.getElementById("reports").children;
                for (let i = 0, child; child = reportsBlockChildren[i]; i++) {
                    if (child.id.match(/report-\d+/)) {
                        child.parentElement.removeChild(child);
                        i--;
                    }
                }
                // Create reports elements
                for (let i = 0; i < data.length; i++) {
                    reports[data[i]] = false;
                    reportBlock = pattern.cloneNode(true);
                    reportBlock.id = 'report-' + (i + 1)
                    reportBlock.className = 'report accordion-item';
                    reportBlock.style.display = 'block';
                    reportBlock.innerHTML = reportBlock.innerHTML.replaceAll("--ID--", i + 1);
                    reportBlock.innerHTML = reportBlock.innerHTML.replaceAll("--DATE--", data[i]);
                    reportBlock.setAttribute("data-report", data[i]);

                    document.getElementById("reports").append(reportBlock);
                    document.getElementById("report-" + (i + 1)).addEventListener("click", loadReport);
                }
            }
            unlockBlock(document.getElementById("reports"));
        }
    };
    xhr.onerror = function () {
        console.log('Error of http connection');
    };
}

function loadReport(event) {
    let reportBlock = event.currentTarget;
    let repDate = reportBlock.dataset.report;
    if (reports[repDate]) {
        return reports[repDate];
    }
    lockBlock(reportBlock);
    let xhr = new XMLHttpRequest();
    xhr.open('POST', '/api/report/' + authToken);
    xhr.setRequestHeader('Content-type', 'application/json; charset=utf-8');
    xhr.send('"' + repDate + '"');
    xhr.onload = function () {
        if (xhr.status != 200) {
            console.log('XHR error: ' + xhr.status);
            return;
        }
        console.log(xhr.response);
        if (xhr.response) {
            reports[repDate] = true;
            let data = JSON.parse(xhr.response);
            let repErrors = data.Errors;
            let keys = Object.keys(repErrors);
            reportBlock.getElementsByClassName('total')[0].innerHTML = 'Total links processed: ' + data.TotalLinks;
            reportBlock.getElementsByClassName('total-errors')[0].innerHTML = 'Total errors: ' + keys.length;
            if (data.URLs) {
                reportBlock.getElementsByClassName('urls-list')[0].innerHTML = '<li>' + data.URLs.join('</li><li>') + '</li>';
            }
            let errBlock = reportBlock.getElementsByClassName("errors")[0];
            if (keys.length > 0) {
                for (var url in repErrors) {
                    errBlock.getElementsByTagName('tbody')[0].innerHTML += '<tr>' +
                        '<td class="text-break"><a href="' + url + '">' + url + '</a></td>' +
                        '<td class="text-break">' + repErrors[url].HTTPStatus + '</td>' +
                        '<td class="text-break">' + repErrors[url].Error + '</td>' +
                        '<td class="text-break"><a href="' + repErrors[url].ParentURL + '">' + repErrors[url].ParentURL + '</a></td>' +
                        '</tr>';
                }
            } else {
                errBlock.parentElement.removeChild(errBlock);
            }
        }
        unlockBlock(reportBlock);
    };
    xhr.onerror = function () {
        console.log('Error of http connection');
    };
}

function lockBlock(block) {
    block.className += ' loading';
}

function unlockBlock(block) {
    block.className = block.className.replaceAll(' loading', '');
}