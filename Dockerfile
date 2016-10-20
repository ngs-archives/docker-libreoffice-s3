FROM alpine:latest
MAINTAINER Atsushi Nagase<a@ngs.io>

RUN apk -Uuv --no-cache add groff less python py-pip libreoffice go curl && \
    pip install awscli && \
    apk --purge -v del py-pip && \
    rm /var/cache/apk/*

RUN curl -Lo /var/tmp/NotoSansCJKjp-hinted.zip \
        https://noto-website.storage.googleapis.com/pkgs/NotoSansCJKjp-hinted.zip && \
        cd /var/tmp && unzip NotoSansCJKjp-hinted.zip && \
        mkdir -p /usr/share/fonts/Type1 && \
        mv *.otf /usr/share/fonts/Type1 && \
        rm -f *

ADD . .

RUN go build -o /usr/bin/convserver convserver.go && rm convserver.go

