FROM ubuntu:focal as client

COPY built_go_server /usr/local/bin/dogfood_server

EXPOSE 51001

ENTRYPOINT [ "/usr/local/bin/dogfood_server" ]
