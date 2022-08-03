#!/usr/bin/env python

import requests
import json

res = requests.get("http://localhost:8080/v1/get/test")
print(res.json())


res = requests.post("http://localhost:8080/v1/post", json={"message":"test"})
print(res.json())

res = requests.get("http://localhost:8080/v1/stream-out/test")
for line in res.iter_lines(chunk_size=None):
    print(json.loads(line))

lines = list( json.dumps({"message": "msg:%d"}) % i for i in range(500) )
res = requests.post(
    "http://localhost:8080/v1/stream-in",
    data= "\n".join(lines)
)
print(res.json())