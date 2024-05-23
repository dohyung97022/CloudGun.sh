const express = require('express');

const app = express();

const PORT = process.env.PORT || 80;

app.get('/', (req, res) => {
    response = {"message": "Hello, world!"}
    res.send(JSON.stringify(response));
});

app.listen(PORT, () => {
    console.log(`Server is running on port ${PORT}`);
});
