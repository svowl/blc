class SocketConnection {

    url;
    conn;
    onMessageHandler;
    prefix = '';

    constructor(url, onMessageHandler) {
        this.url = url;
        this.prefix = url.substring(url.lastIndexOf('/') + 1) + ': ';
        this.onMessageHandler = onMessageHandler;
        this.init();
    }

    init() {
        this.conn = new WebSocket(this.url);

        // событие установки соединения
        this.conn.onopen = () => {
            console.log(this.prefix + "Соединение установлено.");
        };

        // событие разрыва соединения
        this.conn.onclose = (event) => {
            if (event.wasClean) {
                console.log(this.prefix + 'Соединение закрыто чисто');
            } else {
                console.log(this.prefix + 'Обрыв соединения'); // например, "убит" процесс сервера
            }
            console.log(this.prefix + 'Код: ' + event.code + ' причина: ' + event.reason);
        };

        // событие получения сообщения
        this.conn.onmessage = (event) => {
            console.log(this.prefix + "Получены данные " + event.data);
            if (this.onMessageHandler) {
                this.onMessageHandler(event);
            }
        };

        // событие получения ошибки
        this.conn.onerror = (error) => {
            console.log(this.prefix + "Ошибка " + error.message);
        };
    }

    send(data) {
        this.conn.send(data)
        console.log(this.prefix + "Send: " + data)
    }
};
