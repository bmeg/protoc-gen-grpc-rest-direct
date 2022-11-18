#!/usr/bin/env python

import requests
import json

TESTS = [
    "get",
    "post",
    "stream-out",
    "stream-in",
    "post-stream-out"
]

if "get" in TESTS:
    res = requests.get("http://localhost:8080/v1/get/test")
    print(res.json())

if "post" in TESTS:
    res = requests.post("http://localhost:8080/v1/post", json={"message":"test"})
    print(res.json())

if "stream-out" in TESTS:
    res = requests.get("http://localhost:8080/v1/stream-out/test")
    for line in res.iter_lines(chunk_size=None):
        print(json.loads(line))

if "stream-in" in TESTS:
    lines = list( json.dumps({"message": "msg:%d"}) % i for i in range(500) )
    res = requests.post(
        "http://localhost:8080/v1/stream-in",
        data= "\n".join(lines)
    )
    print(res.json())


if "post-stream-out" in TESTS:
    res = requests.post("http://localhost:8080/v1/post-stream-out", json={"message":"test"})
    for line in res.iter_lines(chunk_size=None):
        print(json.loads(line))
