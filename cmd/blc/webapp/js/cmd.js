var cmdConn;

document.addEventListener('authPassed', (event) => {
    console.log('cmd: authPassed event, token: ' + authToken);
    cmdConn = new SocketConnection("ws://" + currentUrl + "/cmd/" + authToken);
    document.getElementById("cmdStart").addEventListener("click", sendCommand);
});

function sendCommand(event) {
    var cmd = event.target.dataset.cmd;
    var data = {
        'Cmd': cmd,
        'URLs': [],
        'Depth': -1,
    }
    if ('start' == cmd) {
        data.URLs = document.getElementById('urls').value.split("\n");
        data.Depth = parseInt(document.getElementById('depthInput').value);
        document.getElementById('startProcessAction').click();
    } else {
        data.ID = parseInt(event.target.dataset.pid);
    }
    cmdConn.send(JSON.stringify(data));
}
