var authToken;

var authEventDispatched = false;

var cookieName = 'authToken';

if (!authToken) {
    console.log('Not authorized')
    if (getCookie(cookieName)) {
        authToken = getCookie(cookieName);
        testToken();
    } else {
        createPopup();
    }

} else {
    dispatchAuthEvent();
}

function dispatchAuthEvent() {
    if (!authEventDispatched) {
        authEventDispatched = true;
        document.getElementsByTagName('body')[0].dispatchEvent(new CustomEvent('authPassed', { bubbles: true }));
        document.getElementById('main').style.display = "block";
    }
}

function getCookie(name) {
    let matches = document.cookie.match(new RegExp(
        "(?:^|; )" + name.replace(/([\.$?*|{}\(\)\[\]\\\/\+^])/g, '\\$1') + "=([^;]*)"
    ));
    return matches ? decodeURIComponent(matches[1]) : undefined;
}

function auth() {
    let xhr = new XMLHttpRequest();
    xhr.open('POST', '/api/signin');
    xhr.setRequestHeader('Content-type', 'application/json; charset=utf-8');
    let data = JSON.stringify({
        'login': document.getElementById('login').value,
        'password': document.getElementById('password').value,
    });
    xhr.send(data);
    xhr.onload = function () {
        if (xhr.status != 200) {
            console.log('XHR error: ' + xhr.status);
            return;
        }
        console.log(xhr.response);
        if (xhr.response) {
            authToken = xhr.response;
            document.cookie = encodeURIComponent(cookieName) + "=" + encodeURIComponent(authToken);
            dispatchAuthEvent();
            let popup = document.getElementById('auth');
            if (popup) {
                popup.style.display = 'none';
                return;
            }
        }
    };
    xhr.onerror = function () {
        console.log('Error of http connection');
    };
}

function testToken() {
    let xhr = new XMLHttpRequest();
    xhr.open('GET', '/api/test/' + authToken);
    xhr.send();
    xhr.onload = function () {
        if (xhr.status != 200) {
            console.log('XHR error: ' + xhr.status);
            return;
        }
        console.log(xhr.response);
        if ("ok" == xhr.response) {
            dispatchAuthEvent();
        } else {
            createPopup();
        }
    };
    xhr.onerror = function () {
        console.log('Error of http connection');
    };
}

function createPopup() {
    let popup = document.getElementById('auth');
    if (popup) {
        popup.style.display = 'block';
        return;
    }
    popup = document.createElement('div');
    popup.id = 'auth';
    popup.className = 'container';
    popup.innerHTML = `
    <div class="mx-auto mt-5 w-50">
        <h3>Sign In</h3>
        <form onsubmit="javascript: auth(); return false;">
            <div class="my-4">
                <input type="text" id="login" class="form-control" placeholder="name@example.com" />
                    </div>
                <div class="my-4">
                    <input type="password" id="password" class="form-control" placeholder="password" />
                </div>
                <button type="submit" class="btn btn-primary">Sign in</button>
        </form>
    </div>
    `;
    document.getElementsByTagName('body')[0].append(popup);
}
