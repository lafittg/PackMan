const express = require('express');
const axios = require('axios');
const _ = require('lodash');

const app = express();

app.get('/api/data', async (req, res) => {
  const response = await axios.get('https://api.example.com/data');
  const sorted = _.sortBy(response.data, 'name');
  res.json(sorted);
});

app.get('/api/users', async (req, res) => {
  const response = await axios.get('https://api.example.com/users');
  const grouped = _.groupBy(response.data, 'role');
  res.json(grouped);
});

app.listen(3000);
