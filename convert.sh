#!/bin/sh

lowriter --invisible --convert-to pdf:writer_pdf_Export --outdir /var/files /var/files/test.pptx
curl -XPOST http://requestb.in/130mvcp1 -H 'Content-Type: text/plain' -d $(ls -lsa /var/files)
