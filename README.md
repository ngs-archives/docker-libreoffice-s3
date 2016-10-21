docker-libreoffice-s3
=====================

[![](https://img.shields.io/docker/automated/atsnngs/libreoffice-s3.svg)](https://hub.docker.com/r/atsnngs/libreoffice-s3/)


Usage
-----

```sh
docker pull atsnngs/libreoffice-s3
docker run -p 8080:8080 \
  -e AWS_REGION=ap-northeast-1 \
  -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
  --rm atsnngs/libreoffice-s3

curl \
  -H 'Content-Type: application/json' \
  -d '{
    "bucket": "my-bucket",
    "key": "/path/to/awesome.pptx",
    "callback_url": "http://requestb.in/xxxxxx",
    "callback_method": "PATCH"
  }' http://0.0.0.0:8080
```

The callback payload would be like:

```json
{
  "status": "completed",
  "thumbnails": {
    "preview": {
      "content_hash": "2bd4e36a5dbd21ea859c44dfbc80f1e4",
      "width": 500,
      "height": 500
    }
  }
}
```

`width` and `height` are fixed number for now :bow:
