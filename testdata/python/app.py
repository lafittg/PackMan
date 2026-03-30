from flask import Flask, jsonify
import requests
from sqlalchemy import create_engine
from celery import Celery
from pydantic import BaseModel
import redis

app = Flask(__name__)

@app.route("/")
def index():
    return jsonify({"status": "ok"})
