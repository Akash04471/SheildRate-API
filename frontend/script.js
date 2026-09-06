function sendRequest() {
    fetch('/api/test')
        .then(response => {
            if (response.status === 429) {
                return response.text().then(text => {
                    throw new Error(text);
                });
            }
            return response.text();
        })
        .then(data => {
            document.getElementById("result").innerText =
                "✅ Allowed: " + data;
        })
        .catch(err => {
            document.getElementById("result").innerText =
                "❌ Blocked: Rate limit exceeded";
        });
}
