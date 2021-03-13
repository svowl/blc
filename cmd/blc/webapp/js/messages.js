const progressStates = {
    1: 'Stopped',
    2: 'In progress',
    3: 'Paused'
}
var processes = [];
var msgConn;
document.addEventListener('authPassed', (event) => {
    console.log('messages: authPassed event, token: ' + authToken);
    msgConn = new SocketConnection(currentProto + currentUrl + "/messages/" + authToken, handleMessage);
});

var handleMessage = function (event) {
    let obj = JSON.parse(event.data);
    let processBlock = document.getElementById('process-' + obj.ID);
    if (obj.ID > 0) {
        if (obj.ProgressState == 1) {
            // Process stopped: remove element and reload reports block
            if (processBlock) {
                processBlock.remove();
            }
            loadReports();
            return;
        }
        // Process process state change
        if (processBlock == null) {
            // Create process block (info, buttons)
            let pattern = document.getElementById("process-pattern");
            if (pattern) {
                processBlock = pattern.cloneNode(true);
                processBlock.id = 'process-' + obj.ID
                processBlock.className = 'process accordion-item status-' + obj.ProgressState;
                processBlock.style.display = 'block';
                processBlock.innerHTML = processBlock.innerHTML.replaceAll("--ID--", obj.ID);
                let buttons = processBlock.getElementsByTagName('button');
                for (let i = 0; i < buttons.length; i++) {
                    if (buttons[i].id) {
                        buttons[i].setAttribute('data-pid', obj.ID);
                        buttons[i].setAttribute('id', buttons[i].id + '-' + obj.ID);
                        buttons[i].addEventListener("click", sendCommand);
                    }
                }
                processBlock.getElementsByClassName('messages')[0].setAttribute('id', 'messages-' + obj.ID);
                processBlock.getElementsByClassName('errors')[0].setAttribute('id', 'errors-' + obj.ID);
                let pInfo = processBlock.getElementsByClassName('process-info')[0];
                pInfo.getElementsByClassName('status')[0].innerHTML = progressStates[obj.ProgressState];

                var currentProcesses = document.getElementById("processes").getElementsByClassName('accordion-button');
                if (currentProcesses.length) {
                    for (i = 0; i < currentProcesses.length; i++) {
                        if (!currentProcesses[i].className.includes('collapsed')) {
                            currentProcesses[i].click();//className += ' collapsed';
                        }
                    }
                }
                document.getElementById("processes").prepend(processBlock);
                loadErrors(obj.ID);
            }
        }
        if (processBlock && obj.URL == "") {
            // Update process info
            processBlock.getElementsByClassName('status')[0].innerHTML = progressStates[obj.ProgressState];
            let statuses = processBlock.classList;
            for (i = 0; i < statuses.length; i++) {
                if (statuses[i].match(/^status.*/)) {
                    statuses.remove(statuses[i]);
                    break;
                }
            }
            statuses.add('status-' + obj.ProgressState);
            let msgBlock = document.getElementById("messages-" + obj.ID);
            if (obj.ProgressState == 1) {
                msgBlock.classList.add('d-none');
            } else {
                msgBlock.classList.remove('d-none');
            }
        }
    }
    if (processBlock) {
        processBlock.getElementsByClassName('total')[0].innerHTML = 'Total links: ' + obj.TotalLinks;
        processBlock.getElementsByClassName('total-errors')[0].innerHTML = 'Total errors: ' + obj.TotalErrors;
        if (obj.URLs) {
            processBlock.getElementsByClassName('urls-list')[0].innerHTML = '<li>' + obj.URLs.join('</li><li>') + '</li>';
        }
    }
    if (obj.URL == "") {
        return
    }
    errClass = 'link-danger';
    if (obj.State == 1) {
        errClass = '';
    } else {
        // Add error message
        let errBlock = document.getElementById("errors-" + obj.ID);
        if (errBlock) {
            errBlock.getElementsByTagName('tbody')[0].innerHTML += '<tr class="process-id-' + obj.ID + '">' +
                '<td class="text-break"><a href="' + obj.URL + '">' + obj.URL + '</a></td>' +
                '<td class="text-break">' + obj.HTTPStatus + '</td>' +
                '<td class="text-break">' + obj.Error + '</td>' +
                '<td class="text-break"><a href="obj.ParentURL">' + obj.ParentURL + '</a></td>' +
                '</tr>';
            if (errBlock.style.display == 'none') {
                errBlock.style.display = 'block';
            }
        }
    }
    // Add new message
    let msgBlock = document.getElementById("messages-" + obj.ID);
    if (msgBlock) {
        msgBlock.innerHTML = '<li class="process-id-' + obj.ID + ' ' + errClass + ' text-break">' + obj.URL + '</li>' + msgBlock.innerHTML;
        if (msgBlock.getElementsByTagName('li').length > 10) {
            msgBlock.removeChild(msgBlock.getElementsByTagName('li')[10]);
        }
    }
}

function loadErrors(id) {
    let xhr = new XMLHttpRequest();
    xhr.open('POST', '/api/processerrors/' + id + '/' + authToken);
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
            if (data.length == 0) return;
            let errBlock = document.getElementById("errors-" + id);
            if (errBlock) {
                for (let url in data) {
                    errBlock.getElementsByTagName('tbody')[0].innerHTML += '<tr class="process-id-' + id + '">' +
                        '<td class="text-break"><a href="' + url + '">' + url + '</a></td>' +
                        '<td class="text-break">' + data[url].HTTPStatus + '</td>' +
                        '<td class="text-break">' + data[url].Error + '</td>' +
                        '<td class="text-break"><a href="' + data[url].ParentURL + '">' + data[url].ParentURL + '</a></td>' +
                        '</tr>';
                    if (errBlock.style.display == 'none') {
                        errBlock.style.display = 'block';
                    }
                }
            }
        }
    };
    xhr.onerror = function () {
        console.log('Error of http connection');
    };
}
